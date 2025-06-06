package instancegroup

import (
	"context"

	scwInstance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
)

// BaseHandler configure the instance server create options with the instance group configuration.
type BaseHandler struct{}

var _ CreateHandler = (*BaseHandler)(nil)

func (h *BaseHandler) Create(_ context.Context, _ *instanceGroup, instance *Instance) error {
	instance.opts = &scwInstance.CreateServerRequest{}
	return nil
}
