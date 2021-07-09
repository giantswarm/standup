package key

import (
	"fmt"
)

const (
	ClusterOwnerName = "conformance-testing"
)

// TaskConfig contains opinionated overrides for certain Tekton tasks (as defined in https://github.com/giantswarm/test-infra).
// For example, sending an AWS release to a China installation when running a China-specific task.
type TaskConfig struct {
	Name string
	// Installation is the name of the management cluster endpoint and kubeconfig to target for this task.
	Installation string
}

// The configurations specified by these TaskConfigs will override the default behavior.
var TaskConfigs = map[string]TaskConfig{
	// generic preserves the default behavior.
	"generic": {
		Name:         "generic",
		Installation: "",
	},
	// aws-china specifies overrides specific to the aws-china task.
	"aws-china": {
		Name:         "aws-china",
		Installation: "aws-china",
	},
}

func KubeconfigPath(base, provider string) (path string) {
	return fmt.Sprintf("%s/%s", base, provider)
}
