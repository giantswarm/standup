package cluster

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var invalidFlagError = &microerror.Error{
	Kind: "invalidFlagError",
}

// IsInvalidFlag asserts invalidFlagError.
func IsInvalidFlag(err error) bool {
	return microerror.Cause(err) == invalidFlagError
}

var notAvailableOrganizationError = &microerror.Error{
	Kind: "notAvailableOrganizationError",
}

// IsNotAvailableOrganization asserts notAvailableOrganizationError.
func IsNotAvailableOrganization(err error) bool {
	return microerror.Cause(err) == notAvailableOrganizationError
}

var notImplementedError = &microerror.Error{
	Kind: "notImplementedError",
}

// IsNotImplemented asserts notImplementedError.
func IsNotImplemented(err error) bool {
	return microerror.Cause(err) == notImplementedError
}

var unsupportedProviderError = &microerror.Error{
	Kind: "unsupportedProviderError",
}

// IsUnsupportedProvider asserts unsupportedProviderError.
func IsUnsupportedProvider(err error) bool {
	return microerror.Cause(err) == unsupportedProviderError
}
