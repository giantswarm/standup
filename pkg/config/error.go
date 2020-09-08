package config

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsClusterCreationError(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}
