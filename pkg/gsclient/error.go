package gsclient

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var clusterCreationError = &microerror.Error{
	Kind: "clusterCreationError",
}

// IsClusterCreationError asserts clusterCreationError.
func IsClusterCreationError(err error) bool {
	return microerror.Cause(err) == clusterCreationError
}

var clusterDeletionError = &microerror.Error{
	Kind: "clusterDeletionError",
}

// IsClusterDeletionError asserts clusterDeletionError.
func IsClusterDeletionError(err error) bool {
	return microerror.Cause(err) == clusterDeletionError
}
