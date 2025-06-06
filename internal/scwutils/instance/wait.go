package scwutils

import (
	"time"

	"errors"

	"github.com/aslafy-z/gitlab-fleeting-plugin-scaleway/internal/scwutils/async"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	scwErrors "github.com/scaleway/scaleway-sdk-go/errors"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	defaultTimeout       = 5 * time.Minute
	defaultRetryInterval = 5 * time.Second
)

// WaitForServerTerminatedRequest is used by WaitForServerTerminated method.
type WaitForServerTerminatedRequest struct {
	ServerID      string
	Zone          scw.Zone
	Timeout       *time.Duration
	RetryInterval *time.Duration
}

// WaitForServerTerminated waits for a server to be terminated.
func WaitForServerTerminated(s *instance.API, req *WaitForServerTerminatedRequest, opts ...scw.RequestOption) error {
	timeout := defaultTimeout
	if req.Timeout != nil {
		timeout = *req.Timeout
	}
	retryInterval := defaultRetryInterval
	if req.RetryInterval != nil {
		retryInterval = *req.RetryInterval
	}

	_, err := async.WaitSync(&async.WaitSyncConfig{
		Get: func() (interface{}, bool, error) {
			_, err := s.GetServer(&instance.GetServerRequest{
				ServerID: req.ServerID,
				Zone:     req.Zone,
			}, opts...)
			if err != nil {
				var notFoundErr *scw.ResourceNotFoundError
				if errors.As(err, &notFoundErr) {
					return true, true, nil
				}
				return false, true, err
			}
			return false, false, err
		},
		Timeout:          timeout,
		IntervalStrategy: async.LinearIntervalStrategy(retryInterval),
	})
	if err != nil {
		return scwErrors.Wrap(err, "waiting for server failed")
	}
	return nil
}
