package wait

import (
	"regexp"

	"github.com/giantswarm/microerror"
)

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

var notReadyError = &microerror.Error{
	Kind: "notReadyError",
}

// IsNotReady asserts notReadyError.
func IsNotReady(err error) bool {
	return microerror.Cause(err) == notReadyError
}

var serverErrorPattern = regexp.MustCompile(`an error on the server .*`)

func IsServerError(err error) bool {
	if err == nil {
		return false
	}
	return serverErrorPattern.MatchString(err.Error())
}
