package key

import (
	"fmt"
)

const (
	ClusterOwnerName = "conformance-testing"
	DefaultTaskName  = "generic"
)

// TaskConfig contains opinionated overrides for certain Tekton tasks (as defined in https://github.com/giantswarm/test-infra).
// For example, sending an AWS release to a China installation when running a China-specific task.
type TaskConfig struct {
	Name string
	// Installation is the name of the management cluster endpoint and kubeconfig to target for this task.
	Installation string
}

// The configurations specified by these TaskConfigs will override the default behavior.
var taskConfigs = map[string]TaskConfig{
	// generic preserves the default behavior.
	DefaultTaskName: {
		Name:         DefaultTaskName,
		Installation: "",
	},
	// aws-china specifies overrides specific to the aws-china task.
	"aws-china": {
		Name:         "aws-china",
		Installation: "aws-china",
	},
}

func GetInstallationForTask(taskName string) string {
	taskConfig, ok := taskConfigs[taskName]
	if ok {
		return taskConfig.Installation
	}

	return ""
}

func GetTaskConfigByName(taskName string) (bool, TaskConfig) {
	taskConfig, ok := taskConfigs[taskName]
	if ok {
		// Return a copy so we don't modify the task globally.
		// Will need to deep copy if we ever have reference types in TaskConfig objects.
		result := taskConfig
		return ok, result
	}

	return ok, TaskConfig{}
}

func KubeconfigPath(base, provider string) (path string) {
	return fmt.Sprintf("%s/%s", base, provider)
}

func TaskConfigs() map[string]TaskConfig {
	// Return a copy to avoid mutating globally.
	return copyMap(taskConfigs)
}

func copyMap(src map[string]TaskConfig) map[string]TaskConfig {
	dst := make(map[string]TaskConfig)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
