package scaleway

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"path"
	"time"

	"github.com/hashicorp/go-hclog"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/sshutil"

	"github.com/aslafy-z/gitlab-fleeting-plugin-scaleway/internal/instancegroup"
	scwBlock "github.com/scaleway/scaleway-sdk-go/api/block/v1"
	scwIam "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	scwInstance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

var _ provider.InstanceGroup = (*InstanceGroup)(nil)

type InstanceGroup struct {
	Name string `json:"name"`

	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key"`
	Organization string `json:"organization"`
	Project      string `json:"project"`
	Endpoint     string `json:"endpoint"`

	Zone         string        `json:"location"`
	ServerTypes  LaxStringList `json:"server_type"`
	Image        string        `json:"image"`
	UserData     string        `json:"user_data"`
	UserDataFile string        `json:"user_data_file"`

	VolumeSize int `json:"volume_size"`

	PublicIPv4Disabled bool `json:"public_ipv4_disabled"`
	PublicIPv6Disabled bool `json:"public_ipv6_disabled"`

	sshKey *scwIam.SSHKey
	tags   []string

	log      hclog.Logger
	settings provider.Settings

	size int

	client         *scw.Client
	instanceClient *scwInstance.API
	iamClient      *scwIam.API
	blockClient    *scwBlock.API

	group instancegroup.InstanceGroup
}

func (g *InstanceGroup) Init(ctx context.Context, log hclog.Logger, settings provider.Settings) (info provider.ProviderInfo, err error) {
	g.settings = settings
	g.log = log.With("zone", g.Zone, "name", g.Name)

	if err = g.validate(); err != nil {
		return
	}

	if err = g.populate(); err != nil {
		return
	}

	// Create client
	clientOptions := []scw.ClientOption{
		scw.WithAuth(g.AccessKey, g.SecretKey),
		scw.WithDefaultOrganizationID(g.Organization),
		scw.WithDefaultProjectID(g.Project),
		scw.WithDefaultZone(scw.Zone(g.Zone)),
		scw.WithUserAgent(fmt.Sprintf("%s (%s)", Version.Name, Version.String())),
		scw.WithHTTPClient(&http.Client{
			Timeout: 15 * time.Second,
		}),
	}
	if g.Endpoint != "" {
		clientOptions = append(clientOptions, scw.WithAPIURL(g.Endpoint))
	}
	g.client, err = scw.NewClient(clientOptions...)
	if err != nil {
		return info, fmt.Errorf("failed to create Scaleway client: %w", err)
	}

	// Prepare credentials
	if !g.settings.UseStaticCredentials {
		g.log.Info("generating ssh key")
		sshPrivateKey, sshPublicKey, err := sshutil.GenerateKeyPair()
		if err != nil {
			return info, err
		}

		g.settings.Key = sshPrivateKey

		g.sshKey, err = g.UploadSSHPublicKey(ctx, sshPublicKey)
		if err != nil {
			return info, err
		}
	} else if len(g.settings.Key) > 0 {
		g.log.Info("using static ssh key")
		sshPublicKey, err := sshutil.GeneratePublicKey(g.settings.Key)
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
		Zone:               g.Zone,
		ServerTypes:        g.ServerTypes,
		Image:              g.Image,
		UserData:           g.UserData,
		Tags:               g.tags,
		VolumeSize:         g.VolumeSize,
		PublicIPv4Disabled: g.PublicIPv4Disabled,
		PublicIPv6Disabled: g.PublicIPv6Disabled,
	}

	g.group = instancegroup.New(g.client, g.log, g.Name, groupConfig)

	if err = g.group.Init(ctx); err != nil {
		return
	}

	return provider.ProviderInfo{
		ID:        path.Join("scaleway", g.Zone, g.Name),
		MaxSize:   math.MaxInt,
		Version:   Version.String(),
		BuildInfo: Version.BuildInfo(),
	}, nil
}

func (g *InstanceGroup) Update(ctx context.Context, update func(string, provider.State)) error {
	instances, err := g.group.List(ctx)
	if err != nil {
		return err
	}

	g.size = len(instances)

	for _, instance := range instances {
		id := instance.IID()

		var state provider.State

		switch instance.Server.State {
		case scwInstance.ServerStateStopping, scwInstance.ServerStateStopped, scwInstance.ServerStateStoppedInPlace:
			state = provider.StateDeleting

		case scwInstance.ServerStateStarting:
			state = provider.StateCreating

		case scwInstance.ServerStateRunning:
			state = provider.StateRunning

		case scwInstance.ServerStateLocked:
			g.log.Debug("unhandled instance status", "id", id, "state", instance.Server.State, "details", instance.Server.StateDetail)
			continue

		default:
			g.log.Error("unexpected instance status", "id", id, "state", instance.Server.State, "details", instance.Server.StateDetail)
			continue
		}

		update(id, state)
	}

	return nil
}

func (g *InstanceGroup) Increase(ctx context.Context, delta int) (int, error) {
	created, err := g.group.Increase(ctx, delta)

	g.size += len(created)

	if sanityErr := g.group.Sanity(ctx); sanityErr != nil {
		g.log.Error("sanity check failed", "error", sanityErr)
	}

	return len(created), err
}

func (g *InstanceGroup) Decrease(ctx context.Context, iids []string) ([]string, error) {
	if len(iids) == 0 {
		return nil, nil
	}

	deleted, err := g.group.Decrease(ctx, iids)

	g.size -= len(deleted)

	if sanityErr := g.group.Sanity(ctx); sanityErr != nil {
		g.log.Error("sanity check failed", "error", sanityErr)
	}

	return deleted, err
}

func (g *InstanceGroup) ConnectInfo(ctx context.Context, iid string) (provider.ConnectInfo, error) {
	info := provider.ConnectInfo{ConnectorConfig: g.settings.ConnectorConfig}

	instance, err := g.group.Get(ctx, iid)
	if err != nil {
		return info, fmt.Errorf("could not get instance: %w", err)
	}

	info.ID = iid
	info.OS = instance.Server.Image.Name

	switch instance.Server.Arch {
	case scwInstance.ArchX86_64:
		info.Arch = "amd64"
	case scwInstance.ArchArm64:
		info.Arch = "arm64"
	default:
		g.log.Warn("unsupported architecture", "architecture", instance.Server.Arch)
	}

	// Handle public IP assignment
	if len(instance.Server.PublicIPs) > 0 && !g.PublicIPv4Disabled {
		for _, publicIP := range instance.Server.PublicIPs {
			if publicIP.Family == scwInstance.ServerIPIPFamilyInet {
				info.ExternalAddr = publicIP.Address.String()
				break
			}
		}
	}

	// Handle IPv6 if no IPv4 found
	if info.ExternalAddr == "" && len(instance.Server.PublicIPs) > 0 && !g.PublicIPv6Disabled {
		for _, publicIP := range instance.Server.PublicIPs {
			if publicIP.Family == scwInstance.ServerIPIPFamilyInet6 {
				info.ExternalAddr = publicIP.Address.String()
				break
			}
		}
	}

	return info, err
}

func (g *InstanceGroup) Heartbeat(_ context.Context, _ string) error {
	// no-op
	return nil
}

func (g *InstanceGroup) Shutdown(ctx context.Context) error {
	errs := make([]error, 0)

	if g.sshKey != nil {
		g.log.Debug("deleting ssh key", "id", fmt.Sprint(g.sshKey.ID))
		err := g.iamClient.DeleteSSHKey(&scwIam.DeleteSSHKeyRequest{
			SSHKeyID: g.sshKey.ID,
		})
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
