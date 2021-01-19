package testoperatorrelease

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

var releaseNotFoundError = &microerror.Error{
	Kind: "releaseNotFoundError",
}

// IsReleaseNotFound asserts releaseNotFoundError.
func IsReleaseNotFound(err error) bool {
	return microerror.Cause(err) == releaseNotFoundError
}

var releaseNotReadyError = &microerror.Error{
	Kind: "releaseNotReadyError",
}

// IsReleaseNotReady asserts releaseNotReadyError.
func IsReleaseNotReady(err error) bool {
	return microerror.Cause(err) == releaseNotReadyError
}
