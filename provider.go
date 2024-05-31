package hetzner

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/hashicorp/go-hclog"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/instancegroup"
	"gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/utils"
)

var _ provider.InstanceGroup = (*InstanceGroup)(nil)

type InstanceGroup struct {
	Name string `json:"name"`

	Token    string `json:"token"`
	Endpoint string `json:"endpoint"`

	Location           string   `json:"location"`
	ServerType         string   `json:"server_type"`
	Image              string   `json:"image"`
	PublicIPv4Disabled bool     `json:"public_ipv4_disabled"`
	PublicIPv6Disabled bool     `json:"public_ipv6_disabled"`
	PrivateNetworks    []string `json:"private_networks"`
	UserData           string   `json:"user_data"`
	UserDataFile       string   `json:"user_data_file"`

	sshKey *hcloud.SSHKey
	labels map[string]string

	log      hclog.Logger
	settings provider.Settings

	size int

	client *hcloud.Client
	group  instancegroup.InstanceGroup
}

func (g *InstanceGroup) Init(ctx context.Context, log hclog.Logger, settings provider.Settings) (info provider.ProviderInfo, err error) {
	g.settings = settings
	g.log = log.With("location", g.Location, "name", g.Name)

	if err = g.validate(); err != nil {
		return
	}

	if err = g.populate(); err != nil {
		return
	}

	// Create client
	clientOptions := []hcloud.ClientOption{
		hcloud.WithApplication(Version.Name, Version.String()),
		hcloud.WithToken(g.Token),
		hcloud.WithHTTPClient(&http.Client{
			Timeout: 15 * time.Second,
		}),
	}
	if g.Endpoint != "" {
		clientOptions = append(clientOptions, hcloud.WithEndpoint(g.Endpoint))
	}
	g.client = hcloud.NewClient(clientOptions...)

	// Prepare credentials
	if !g.settings.UseStaticCredentials {
		g.log.Info("generating new ssh key")
		sshPrivateKey, sshPublicKey, err := utils.GenerateSSHKeyPair()
		if err != nil {
			return info, err
		}

		g.settings.Key = sshPrivateKey

		g.sshKey, err = g.UploadSSHPublicKey(ctx, sshPublicKey)
		if err != nil {
			return info, err
		}
	} else if len(g.settings.Key) > 0 {
		g.log.Info("using configured static ssh key")
		sshPublicKey, err := utils.GenerateSSHPublicKey(g.settings.Key)
		if err != nil {
			return info, err
		}

		g.sshKey, err = g.UploadSSHPublicKey(ctx, sshPublicKey)
		if err != nil {
			return info, err
		}
	}

	// Create instance group
	groupConfig := instancegroup.Config{
		Location:           g.Location,
		ServerType:         g.ServerType,
		Image:              g.Image,
		PublicIPv4Disabled: g.PublicIPv4Disabled,
		PublicIPv6Disabled: g.PublicIPv6Disabled,
		PrivateNetworks:    g.PrivateNetworks,
		UserData:           g.UserData,
		Labels:             g.labels,
	}

	if g.sshKey != nil {
		groupConfig.SSHKeys = []string{g.sshKey.Name}
	}

	g.group = instancegroup.New(g.client, g.Name, groupConfig)

	if err = g.group.Init(ctx); err != nil {
		return
	}

	return provider.ProviderInfo{
		ID:        path.Join("hetzner", g.Location, g.ServerType, g.Name),
		MaxSize:   math.MaxInt,
		Version:   Version.String(),
		BuildInfo: Version.BuildInfo(),
	}, nil
}

func (g *InstanceGroup) Update(ctx context.Context, update func(id string, state provider.State)) error {
	instances, err := g.group.List(ctx)
	if err != nil {
		return err
	}

	g.size = len(instances)

	for _, instance := range instances {
		var state provider.State

		switch instance.Status {
		case hcloud.ServerStatusStopping, hcloud.ServerStatusDeleting:
			state = provider.StateDeleting

		// Server creation always go through `initializing` and `off`. Since we never
		// shutdown servers, we can assume that "off" is still in the creation phase.
		case hcloud.ServerStatusOff:
			state = provider.StateCreating

		case hcloud.ServerStatusInitializing, hcloud.ServerStatusStarting:
			state = provider.StateCreating

		case hcloud.ServerStatusRunning:
			state = provider.StateRunning

		case hcloud.ServerStatusMigrating, hcloud.ServerStatusRebuilding, hcloud.ServerStatusUnknown:
			g.log.Debug("unhandled instance status", "id", instance.ID, "status", instance.Status)

		default:
			g.log.Error("unexpected instance status", "id", instance.ID, "status", instance.Status)
		}

		update(strconv.FormatInt(instance.ID, 10), state)
	}

	return nil
}

func (g *InstanceGroup) Increase(ctx context.Context, delta int) (int, error) {
	created, err := g.group.Increase(ctx, delta)

	g.size += len(created)

	return len(created), err
}

func (g *InstanceGroup) Decrease(ctx context.Context, instances []string) ([]string, error) {
	if len(instances) == 0 {
		return nil, nil
	}

	ids, err := utils.ParseIDList(instances)
	if err != nil {
		return nil, err
	}

	errs := make([]error, 0)

	deleted, err := g.group.Decrease(ctx, ids)
	if err != nil {
		errs = append(errs, err)
	}

	g.size -= len(deleted)

	return utils.FormatIDList(deleted), errors.Join(errs...)
}

func (g *InstanceGroup) ConnectInfo(ctx context.Context, instance string) (provider.ConnectInfo, error) {
	info := provider.ConnectInfo{ConnectorConfig: g.settings.ConnectorConfig}

	id, err := utils.ParseID(instance)
	if err != nil {
		return info, fmt.Errorf("could not parse instance id: %w", err)
	}

	server, err := g.group.Get(ctx, id)
	if err != nil {
		return info, fmt.Errorf("could not get instance: %w", err)
	}

	info.ID = instance
	info.OS = server.Image.OSFlavor

	switch server.ServerType.Architecture {
	case hcloud.ArchitectureX86:
		info.Arch = "amd64"
	case hcloud.ArchitectureARM:
		info.Arch = "arm64"
	default:
		g.log.Warn("unsupported architecture", "architecture", server.ServerType.Architecture)
	}

	switch {
	case !server.PublicNet.IPv4.IsUnspecified():
		info.ExternalAddr = server.PublicNet.IPv4.IP.String()
	case !server.PublicNet.IPv6.IsUnspecified():
		info.ExternalAddr = server.PublicNet.IPv6.IP.String()
	}

	if len(server.PrivateNet) > 0 {
		info.InternalAddr = server.PrivateNet[0].IP.String()
	}

	return info, err
}

func (g *InstanceGroup) Shutdown(ctx context.Context) error {
	errs := make([]error, 0)

	if g.sshKey != nil {
		g.log.Debug("deleting ssh key", "id", g.sshKey.ID)
		_, err := g.client.SSHKey.Delete(ctx, g.sshKey)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
