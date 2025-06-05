package instancegroup

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/hashicorp/go-hclog"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/randutil"
	scwBlock "github.com/scaleway/scaleway-sdk-go/api/block/v1"
	scwInstance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type InstanceGroup interface {
	Init(ctx context.Context) error

	Increase(ctx context.Context, delta int) ([]string, error)
	Decrease(ctx context.Context, iids []string) ([]string, error)

	List(ctx context.Context) ([]*Instance, error)
	Get(ctx context.Context, iid string) (*Instance, error)

	Sanity(ctx context.Context) error
}

var _ InstanceGroup = (*instanceGroup)(nil)

func New(client *scw.Client, log hclog.Logger, name string, config Config) InstanceGroup {
	return &instanceGroup{
		name:           name,
		config:         config,
		log:            log,
		instanceClient: scwInstance.NewAPI(client),
		blockClient:    scwBlock.NewAPI(client),
	}
}

type instanceGroup struct {
	name   string
	config Config

	// TODO: Replace with slog once https://github.com/hashicorp/go-hclog/pull/144 is
	// merged.
	log            hclog.Logger
	instanceClient *scwInstance.API
	blockClient    *scwBlock.API

	zone        *scw.Zone
	serverTypes []string
	image       *string
	tags        []string

	randomNameFn func() string
}

func (g *instanceGroup) Init(ctx context.Context) (err error) {
	if g.randomNameFn == nil {
		g.randomNameFn = func() string {
			return g.name + "-" + randutil.GenerateID()
		}
	}

	// Location
	zone := scw.Zone(g.config.Zone)
	if !zone.Exists() {
		return fmt.Errorf("zone not found: %s", g.config.Zone)
	}
	g.zone = &zone

	// Server Type
	for _, serverTypeID := range g.config.ServerTypes {
		_, err := g.instanceClient.GetServerType(
			&scwInstance.GetServerTypeRequest{
				Zone: *g.zone,
				Name: serverTypeID,
			},
		)
		if err != nil {
			return fmt.Errorf("server type not found: %s: %w", serverTypeID, err)
		}
		g.serverTypes = append(g.serverTypes, serverTypeID)
	}

	// Image
	if _, err := g.instanceClient.GetImage(
		&scwInstance.GetImageRequest{
			Zone:    *g.zone,
			ImageID: g.config.Image,
		},
		scw.WithContext(ctx),
	); err != nil {
		return fmt.Errorf("image not found: %s: %w", g.config.Image, err)
	}
	g.image = &g.config.Image

	g.tags = make([]string, 0, len(g.config.Tags))
	if g.config.Tags != nil {
		g.tags = append(g.tags, g.config.Tags...)
	}
	g.tags = append(g.tags, fmt.Sprintf("instance-group=%s", g.name))

	// If the server name prefix is not set, use the instance group name as the prefix.
	if g.config.ServerNamePrefix == "" {
		g.config.ServerNamePrefix = g.name
	}

	return nil
}

func (g *instanceGroup) Increase(ctx context.Context, delta int) ([]string, error) {
	handlers := []CreateHandler{
		&BaseHandler{},   // Configure the instance server create options from the instance group config.
		&ServerHandler{}, // Create a server from the instance server create options.
	}

	// Run all pre increase handlers
	for _, handler := range handlers {
		h, ok := handler.(PreIncreaseHandler)
		if !ok {
			continue
		}

		if err := h.PreIncrease(ctx, g); err != nil {
			return nil, err
		}
	}

	errs := make([]error, 0)

	instances := make([]*Instance, 0, delta)
	failed := make([]*Instance, 0, delta)

	// Create a list of new instances
	for i := 0; i < delta; i++ {
		instances = append(instances, NewInstance(g.randomNameFn()))
	}

	// Run all create handlers on each instance
	for _, handler := range handlers {
		{
			succeeded := make([]*Instance, 0, len(instances))
			for _, instance := range instances {
				if err := handler.Create(ctx, g, instance); err != nil {
					errs = append(errs, err)
					failed = append(failed, instance)
				} else {
					succeeded = append(succeeded, instance)
				}
			}
			instances = succeeded
		}

		// Wait for each instance background tasks to complete
		{
			succeeded := make([]*Instance, 0, len(instances))
			for _, instance := range instances {
				if err := instance.wait(); err != nil {
					errs = append(errs, err)
					failed = append(failed, instance)
				} else {
					succeeded = append(succeeded, instance)
				}
			}
			instances = succeeded
		}
	}

	// Cleanup failed instances
	if len(failed) > 0 {
		// During cleanup, the handlers must be run backwards
		slices.Reverse(handlers)

		// Run all cleanup handlers on each failed instance
		for _, handler := range handlers {
			h, ok := handler.(CleanupHandler)
			if !ok {
				continue
			}

			for _, instance := range failed {
				if err := h.Cleanup(ctx, g, instance); err != nil {
					errs = append(errs, err)
				}
			}

			// Wait for each instance background tasks to complete
			for _, instance := range failed {
				if err := instance.wait(); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	// Collect created instances IIDs
	created := make([]string, 0, len(instances))
	for _, instance := range instances {
		created = append(created, instance.IID())
	}

	return created, errors.Join(errs...)
}

func (g *instanceGroup) Decrease(ctx context.Context, iids []string) ([]string, error) {
	handlers := []CleanupHandler{
		&ServerHandler{}, // Delete the server of the instance.
	}

	// Run all pre decrease handlers
	for _, handler := range handlers {
		h, ok := handler.(PreDecreaseHandler)
		if !ok {
			continue
		}

		if err := h.PreDecrease(ctx, g); err != nil {
			return nil, err
		}
	}

	errs := make([]error, 0)

	instances := make([]*Instance, 0, len(iids))

	// Populate a list of instances from their IIDs
	for _, iid := range iids {
		instance, err := InstanceFromIID(iid)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		instances = append(instances, instance)
	}

	// Run all cleanup handlers on each instance
	for _, handler := range handlers {
		{
			succeeded := make([]*Instance, 0, len(instances))
			for _, instance := range instances {
				if err := handler.Cleanup(ctx, g, instance); err != nil {
					errs = append(errs, err)
				} else {
					succeeded = append(succeeded, instance)
				}
			}
			instances = succeeded
		}

		// Wait for each instance background tasks to complete
		{
			succeeded := make([]*Instance, 0, len(instances))
			for _, instance := range instances {
				if err := instance.wait(); err != nil {
					errs = append(errs, err)
				} else {
					succeeded = append(succeeded, instance)
				}
			}
			instances = succeeded
		}
	}

	// Collect deleted instances IIDs
	deleted := make([]string, 0, len(instances))
	for _, instance := range instances {
		deleted = append(deleted, instance.IID())
	}

	return deleted, errors.Join(errs...)
}

func (g *instanceGroup) List(ctx context.Context) ([]*Instance, error) {
	servers, err := g.instanceClient.ListServers(
		&scwInstance.ListServersRequest{
			Zone: *g.zone,
			Tags: g.tags,
		},
		scw.WithAllPages(),
		scw.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("could not list instances: %w", err)
	}

	instances := make([]*Instance, 0, len(servers.Servers))
	for _, server := range servers.Servers {
		instances = append(instances, InstanceFromServer(server))
	}

	return instances, nil
}

func (g *instanceGroup) Get(ctx context.Context, iid string) (*Instance, error) {
	instance, err := InstanceFromIID(iid)
	if err != nil {
		return nil, err
	}

	server, err := g.instanceClient.GetServer(
		&scwInstance.GetServerRequest{
			Zone:     *g.zone,
			ServerID: instance.IID(),
		},
		scw.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("could not get instance: %w", err)
	}

	return InstanceFromServer(server.Server), nil
}

func (g *instanceGroup) Sanity(ctx context.Context) error {
	handlers := []SanityHandler{}

	// Run all sanity handlers
	for _, h := range handlers {
		if err := h.Sanity(ctx, g); err != nil {
			g.log.With("handler", reflect.TypeOf(h).String()).Error(err.Error())
		}
	}

	return nil
}
