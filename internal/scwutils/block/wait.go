package scwutils

import (
	"time"

	"github.com/aslafy-z/gitlab-fleeting-plugin-scaleway/internal/scwutils/async"
	"github.com/scaleway/scaleway-sdk-go/api/block/v1"
	"github.com/scaleway/scaleway-sdk-go/errors"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	defaultTimeout       = 5 * time.Minute
	defaultRetryInterval = 5 * time.Second
)

// WaitForServerRequest is used by WaitForServer method.
type WaitForVolumeStatusRequest struct {
	VolumeID      string
	Zone          scw.Zone
	Timeout       *time.Duration
	RetryInterval *time.Duration
	Statuses      []block.VolumeStatus
}

// WaitForVolumeStatus waits for a volume to reach one of the desired statuses.
// This function can be used to wait for a volume to be unattached for example.
func WaitForVolumeStatus(s *block.API, req *WaitForVolumeStatusRequest, opts ...scw.RequestOption) (*block.Volume, error) {
	timeout := defaultTimeout
	if req.Timeout != nil {
		timeout = *req.Timeout
	}
	retryInterval := defaultRetryInterval
	if req.RetryInterval != nil {
		retryInterval = *req.RetryInterval
	}

	volume, err := async.WaitSync(&async.WaitSyncConfig{
		Get: func() (interface{}, bool, error) {
			volume, err := s.GetVolume(&block.GetVolumeRequest{
				VolumeID: req.VolumeID,
				Zone:     req.Zone,
			}, opts...)
			if err != nil {
				return nil, false, err
			}

			isReached := false
			for _, status := range req.Statuses {
				if volume.Status == status {
					isReached = true
					break
				}
			}

			return volume, isReached, err
		},
		Timeout:          timeout,
		IntervalStrategy: async.LinearIntervalStrategy(retryInterval),
	})
	if err != nil {
		return nil, errors.Wrap(err, "waiting for volume status failed")
	}
	return volume.(*block.Volume), nil
}
