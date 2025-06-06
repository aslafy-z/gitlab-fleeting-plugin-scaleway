package instancegroup

import (
	"context"
	"fmt"
	"strings"

	scwBlockUtils "github.com/aslafy-z/gitlab-fleeting-plugin-scaleway/internal/scwutils/block"
	scwErrors "github.com/aslafy-z/gitlab-fleeting-plugin-scaleway/internal/scwutils/errors"
	scwInstanceUtils "github.com/aslafy-z/gitlab-fleeting-plugin-scaleway/internal/scwutils/instance"
	scwBlock "github.com/scaleway/scaleway-sdk-go/api/block/v1"
	scwInstance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// ServerHandler creates a server from the instance server create options.
type ServerHandler struct{}

var _ CreateHandler = (*ServerHandler)(nil)
var _ CleanupHandler = (*ServerHandler)(nil)

func (h *ServerHandler) Create(ctx context.Context, group *instanceGroup, instance *Instance) error {
	tags := append([]string{fmt.Sprintf("instance=%s", instance.Name)}, group.tags...)

	instance.opts.Name = instance.Name
	instance.opts.Tags = tags
	instance.opts.Zone = *group.zone
	instance.opts.Image = &group.image
	instance.opts.DynamicIPRequired = scw.BoolPtr(false)

	var err error

	//// TODO: Choose between two options:
	// // 1. Create a volume from an image -> needs extracting snapshot ID from the image
	// // 2. Let Scaleway create the volume from the image automatically and patch the name, tags, and IOPS later
	// // For now, we will use option 2, as it is simpler and works with all images.
	// // With option 2, if some error occurs, it may be tedious to identify the volume that should be delete.

	// volumeRes, err := group.blockClient.CreateVolume(
	// 	&scwBlock.CreateVolumeRequest{
	// 		Name:     scw.StringPtr(fmt.Sprintf("%s-os", instance.Name)),
	// 		Tags:     group.tags,
	// 		PerfIops: scw.Uint32Ptr(5000), // TODO: Make this configurable
	// 		FromEmpty: &scwBlock.CreateVolumeRequestFromEmpty{
	// 			Size: scw.Size(group.config.VolumeSize * 1000 * 1000 * 1000), // Size in bytes
	// 		},
	// 		Zone: *group.zone,
	// 	},
	// 	scw.WithContext(ctx),
	// )
	// if err != nil {
	// 	return fmt.Errorf("could not create volume: %w", err)
	// }

	instance.opts.Volumes = map[string]*scwInstance.VolumeServerTemplate{
		"0": {
			// ID:         &volumeRes.ID,
			// Boot:       scw.BoolPtr(true),
			VolumeType: scwInstance.VolumeVolumeTypeSbsVolume,
			// Name:       scw.StringPtr(fmt.Sprintf("%s-os", instance.Name)),
		},
	}

	var publicIPs []string
	if !group.config.PublicIPv4Disabled {
		ipRes, err := group.instanceClient.CreateIP(
			&scwInstance.CreateIPRequest{
				Tags: tags,
				Type: scwInstance.IPTypeRoutedIPv4,
				Zone: *group.zone,
			},
			scw.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("could not create IPv4: %w", err)
		}
		publicIPs = append(publicIPs, ipRes.IP.ID)
	}
	if !group.config.PublicIPv6Disabled {
		ipRes, err := group.instanceClient.CreateIP(
			&scwInstance.CreateIPRequest{
				Tags: tags,
				Type: scwInstance.IPTypeRoutedIPv6,
				Zone: *group.zone,
			},
			scw.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("could not create IPv6: %w", err)
		}
		publicIPs = append(publicIPs, ipRes.IP.ID)
	}
	instance.opts.PublicIPs = &publicIPs

	var result *scwInstance.CreateServerResponse

	for _, serverType := range group.serverTypes {
		instance.opts.CommercialType = serverType
		result, err = group.instanceClient.CreateServer(
			instance.opts,
			scw.WithContext(ctx),
		)
		if scwErrors.IsOutOfStockError(err) {
			group.log.Warn("server type not available", "server_type", serverType, "error", err)
			continue
		}

		break
	}
	if err != nil {
		return fmt.Errorf("could not request instance creation: %w", err)
	}

	_, err = group.blockClient.UpdateVolume(
		&scwBlock.UpdateVolumeRequest{
			VolumeID: result.Server.Volumes["0"].ID,
			Name:     scw.StringPtr(fmt.Sprintf("%s-os", instance.Name)),
			Tags:     &tags,
			PerfIops: scw.Uint32Ptr(5000), // TODO: Make this configurable
		},
		scw.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("could not update volume name: %w", err)
	}

	err = group.instanceClient.SetServerUserData(
		&scwInstance.SetServerUserDataRequest{
			ServerID: result.Server.ID,
			Key:      "cloud-init",
			Content:  strings.NewReader(group.config.CloudInit),
		},
		scw.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("could not set server user data: %w", err)
	}

	_, err = group.instanceClient.ServerAction(
		&scwInstance.ServerActionRequest{
			ServerID: result.Server.ID,
			Action:   scwInstance.ServerActionPoweron,
		},
		scw.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("could not power on server: %w", err)
	}

	*instance = *InstanceFromServer(result.Server)

	instance.waitFn = func() error {
		_, err = group.instanceClient.WaitForServer(
			&scwInstance.WaitForServerRequest{
				ServerID: result.Server.ID,
			},
			scw.WithContext(ctx),
		)

		return err
	}

	return nil
}

func (h *ServerHandler) Cleanup(ctx context.Context, group *instanceGroup, instance *Instance) error {
	if instance.ID == "" {
		return nil
	}

	// TODO: Prevent repetition
	tags := []string{fmt.Sprintf("instance=%s", instance.Name)}

	// Delete all public IPs associated with the server
	ipsRes, err := group.instanceClient.ListIPs(
		&scwInstance.ListIPsRequest{
			Tags: tags,
			Zone: *group.zone,
		},
		scw.WithAllPages(),
		scw.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("could not list IPs: %w", err)
	}
	for _, ip := range ipsRes.IPs {
		err = group.instanceClient.DeleteIP(
			&scwInstance.DeleteIPRequest{
				IP: ip.ID,
			},
			scw.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("could not delete IP: %w", err)
		}
	}

	serverRes, err := group.instanceClient.GetServer(
		&scwInstance.GetServerRequest{
			ServerID: instance.ID,
		},
		scw.WithContext(ctx),
	)

	if err != nil && !scwErrors.IsResourceNotFoundError(err) {
		return fmt.Errorf("could not get server: %w", err)
	}

	serverPresent := false
	if serverRes != nil && serverRes.Server != nil {
		serverPresent = true
	}

	if serverPresent {
		err = group.instanceClient.ServerActionAndWait(
			&scwInstance.ServerActionAndWaitRequest{
				ServerID: instance.ID,
				Action:   scwInstance.ServerActionPoweroff,
			},
			scw.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("could not power off server: %w", err)
		}

		if len(serverRes.Server.Volumes) > 0 {
			// Detach all volumes attached to the server
			for _, volume := range serverRes.Server.Volumes {
				_, err = group.instanceClient.DetachServerVolume(
					&scwInstance.DetachServerVolumeRequest{
						ServerID: instance.ID,
						VolumeID: volume.ID,
					},
					scw.WithContext(ctx),
				)
				if err != nil {
					return fmt.Errorf("could not detach volume: %w", err)
				}
			}
		}
	}

	volumesRes, err := group.blockClient.ListVolumes(
		&scwBlock.ListVolumesRequest{
			Tags: tags,
			Zone: *group.zone,
		},
		scw.WithAllPages(),
		scw.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("could not list volumes: %w", err)
	}

	for _, volume := range volumesRes.Volumes {
		_, err = scwBlockUtils.WaitForVolumeStatus(
			group.blockClient,
			&scwBlockUtils.WaitForVolumeStatusRequest{
				VolumeID: volume.ID,
				Statuses: []scwBlock.VolumeStatus{scwBlock.VolumeStatusAvailable},
			},
			scw.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("could not wait for volume to be detached: %w", err)
		}

		err = group.blockClient.DeleteVolume(
			&scwBlock.DeleteVolumeRequest{
				VolumeID: volume.ID,
			},
			scw.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("could not delete volume: %w", err)
		}
	}

	if serverPresent {
		err = group.instanceClient.DeleteServer(
			&scwInstance.DeleteServerRequest{
				ServerID: instance.ID,
			},
			scw.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("could not request instance deletion: %w", err)
		}

		instance.waitFn = func() error {
			err = scwInstanceUtils.WaitForServerTerminated(
				group.instanceClient,
				&scwInstanceUtils.WaitForServerTerminatedRequest{
					ServerID: instance.ID,
				},
				scw.WithContext(ctx),
			)
			if err != nil {
				return fmt.Errorf("could not wait for server deletion: %w", err)
			}
			return nil
		}
	}

	return nil
}
