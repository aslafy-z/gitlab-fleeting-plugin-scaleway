package scw

import (
	"errors"

	"github.com/scaleway/scaleway-sdk-go/scw"
)

// IsResponseError checks if an error is a ResponseError.
func IsResponseError(err error) bool {
	var target *scw.ResponseError
	return errors.As(err, &target)
}

// IsInvalidArgumentsError checks if an error is an InvalidArgumentsError.
func IsInvalidArgumentsError(err error) bool {
	var target *scw.InvalidArgumentsError
	return errors.As(err, &target)
}

// IsQuotasExceededError checks if an error is a QuotasExceededError.
func IsQuotasExceededError(err error) bool {
	var target *scw.QuotasExceededError
	return errors.As(err, &target)
}

// IsPermissionsDeniedError checks if an error is a PermissionsDeniedError.
func IsPermissionsDeniedError(err error) bool {
	var target *scw.PermissionsDeniedError
	return errors.As(err, &target)
}

// IsTransientStateError checks if an error is a TransientStateError.
func IsTransientStateError(err error) bool {
	var target *scw.TransientStateError
	return errors.As(err, &target)
}

// IsResourceNotFoundError checks if an error is a ResourceNotFoundError.
func IsResourceNotFoundError(err error) bool {
	var target *scw.ResourceNotFoundError
	return errors.As(err, &target)
}

// IsResourceLockedError checks if an error is a ResourceLockedError.
func IsResourceLockedError(err error) bool {
	var target *scw.ResourceLockedError
	return errors.As(err, &target)
}

// IsOutOfStockError checks if an error is an OutOfStockError.
func IsOutOfStockError(err error) bool {
	var target *scw.OutOfStockError
	return errors.As(err, &target)
}

// IsInvalidClientOptionError checks if an error is an InvalidClientOptionError.
func IsInvalidClientOptionError(err error) bool {
	var target *scw.InvalidClientOptionError
	return errors.As(err, &target)
}

// IsConfigFileNotFoundError checks if an error is a ConfigFileNotFoundError.
func IsConfigFileNotFoundError(err error) bool {
	var target *scw.ConfigFileNotFoundError
	return errors.As(err, &target)
}

// IsResourceExpiredError checks if an error is a ResourceExpiredError.
func IsResourceExpiredError(err error) bool {
	var target *scw.ResourceExpiredError
	return errors.As(err, &target)
}

// IsDeniedAuthenticationError checks if an error is a DeniedAuthenticationError.
func IsDeniedAuthenticationError(err error) bool {
	var target *scw.DeniedAuthenticationError
	return errors.As(err, &target)
}

// IsPreconditionFailedError checks if an error is a PreconditionFailedError.
func IsPreconditionFailedError(err error) bool {
	var target *scw.PreconditionFailedError
	return errors.As(err, &target)
}
