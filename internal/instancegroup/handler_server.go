package instancegroup

import (
	"context"
	"fmt"
	"strings"

	scwBlock "github.com/scaleway/scaleway-sdk-go/api/block/v1"
	scwInstance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// ServerHandler creates a server from the instance server create options.
type ServerHandler struct{}

var _ CreateHandler = (*ServerHandler)(nil)
var _ CleanupHandler = (*ServerHandler)(nil)

func (h *ServerHandler) Create(ctx context.Context, group *instanceGroup, instance *Instance) error {
	instance.opts.Name = instance.Name
	instance.opts.Tags = group.tags
	instance.opts.Zone = *group.zone
	instance.opts.Image = group.image
	instance.opts.DynamicIPRequired = scw.BoolPtr(false)
	instance.opts.Volumes = map[string]*scwInstance.VolumeServerTemplate{
		"0": {
			Size:       scw.SizePtr(scw.Size(group.config.VolumeSize)),
			VolumeType: scwInstance.VolumeVolumeTypeSbsVolume,
		},
	}

	var result *scwInstance.CreateServerResponse
	var err error

	var publicIPs []string
	if !group.config.PublicIPv4Disabled {
		ipRes, err := group.instanceClient.CreateIP(&scwInstance.CreateIPRequest{
			Tags: group.tags,
			Type: scwInstance.IPTypeRoutedIPv4,
		}, scw.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("could not create IPv4: %w", err)
		}
		publicIPs = append(publicIPs, ipRes.IP.ID)
	}
	if !group.config.PublicIPv6Disabled {
		ipRes, err := group.instanceClient.CreateIP(&scwInstance.CreateIPRequest{
			Tags: group.tags,
			Type: scwInstance.IPTypeRoutedIPv6,
		}, scw.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("could not create IPv6: %w", err)
		}
		publicIPs = append(publicIPs, ipRes.IP.ID)
	}
	instance.opts.PublicIPs = &publicIPs

	for _, serverType := range group.serverTypes {
		srvAvailabilityRes, err := group.instanceClient.GetServerTypesAvailability(&scwInstance.GetServerTypesAvailabilityRequest{}, scw.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("could not check server type availability: %w", err)
		}
		srvAvailability, exists := srvAvailabilityRes.Servers[serverType]
		if !exists || srvAvailability.Availability != scwInstance.ServerTypesAvailabilityAvailable {
			group.log.Warn("server type not available", "server_type", serverType)
			continue
		}

		instance.opts.CommercialType = serverType
		result, err = group.instanceClient.CreateServer(instance.opts)
		if err != nil {
			return fmt.Errorf("could not request instance creation: %w", err)
		}
		break
	}

	err = group.instanceClient.SetServerUserData(&scwInstance.SetServerUserDataRequest{
		ServerID: result.Server.ID,
		Key:      "cloud-init",
		Content:  strings.NewReader(group.config.UserData),
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("could not set server user data: %w", err)
	}

	_, err = group.blockClient.UpdateVolume(&scwBlock.UpdateVolumeRequest{
		VolumeID: result.Server.Volumes["0"].ID,
		PerfIops: scw.Uint32Ptr(5000), // TODO: Make this configurable
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("could not update volume IOPS: %w", err)
	}

	_, err = group.instanceClient.ServerAction(&scwInstance.ServerActionRequest{
		ServerID: result.Server.ID,
		Action:   scwInstance.ServerActionPoweron,
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("could not power on server: %w", err)
	}

	// TODO: Fix image iD
	*instance = *InstanceFromServer(result.Server)

	instance.waitFn = func() error {
		_, err = group.instanceClient.WaitForServer(&scwInstance.WaitForServerRequest{
			ServerID: result.Server.ID,
		}, scw.WithContext(ctx))

		return err
	}

	return nil
}

func (h *ServerHandler) Cleanup(ctx context.Context, group *instanceGroup, instance *Instance) error {
	// TODO: Why
	// if instance.ID == 0 {
	// 	return nil
	// }

	err := group.instanceClient.DeleteIP(&scwInstance.DeleteIPRequest{
		IP: instance.Server.PublicIPs[0].ID,
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("could not delete IP: %w", err)
	}

	err = group.instanceClient.ServerActionAndWait(&scwInstance.ServerActionAndWaitRequest{
		ServerID: instance.ID,
		Action:   scwInstance.ServerActionPoweroff,
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("could not power off server: %w", err)
	}

	_, err = group.instanceClient.DetachServerVolume(&scwInstance.DetachServerVolumeRequest{
		ServerID: instance.ID,
		VolumeID: instance.Server.Volumes["0"].ID,
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("could not detach volume: %w", err)
	}

	err = group.instanceClient.DeleteVolume(&scwInstance.DeleteVolumeRequest{
		VolumeID: instance.Server.Volumes["0"].ID,
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("could not delete volume: %w", err)
	}

	err = group.instanceClient.DeleteServer(&scwInstance.DeleteServerRequest{
		ServerID: instance.ID,
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("could not request instance deletion: %w", err)
	}

	instance.waitFn = func() error {
		_, err = group.instanceClient.WaitForServer(&scwInstance.WaitForServerRequest{
			ServerID: instance.ID,
		}, scw.WithContext(ctx))

		return err
	}

	return nil
}
