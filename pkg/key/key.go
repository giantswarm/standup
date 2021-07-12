package key

import (
	"fmt"
)

const (
	ClusterOwnerName    = "conformance-testing"
	DefaultPipelineName = "generic"
)

// PipelineConfig contains opinionated overrides for certain Tekton pipelines (as defined in https://github.com/giantswarm/test-infra).
// For example, sending an AWS release to a China installation when running a China-specific pipeline.
type PipelineConfig struct {
	Name string
	// Installation is the name of the management cluster endpoint and kubeconfig to target for this pipeline.
	Installation string
}

// The configurations specified by these PipelineConfigs will override the default behavior.
var pipelineConfigs = map[string]PipelineConfig{
	// generic preserves the default behavior.
	DefaultPipelineName: {
		Name:         DefaultPipelineName,
		Installation: "",
	},
	// aws-china specifies overrides specific to the aws-china pipeline.
	"aws-china": {
		Name:         "aws-china",
		Installation: "aws-china",
	},
}

func GetInstallationForPipeline(pipelineName string) string {
	pipelineConfig, ok := pipelineConfigs[pipelineName]
	if ok {
		return pipelineConfig.Installation
	}

	return ""
}

func GetPipelineConfigByName(pipelineName string) (bool, PipelineConfig) {
	pipelineConfig, ok := pipelineConfigs[pipelineName]
	if ok {
		// Return a copy so we don't modify the pipeline globally.
		// Will need to deep copy if we ever have reference types in PipelineConfig objects.
		result := pipelineConfig
		return ok, result
	}

	return ok, PipelineConfig{}
}

func KubeconfigPath(base, provider string) (path string) {
	return fmt.Sprintf("%s/%s", base, provider)
}

func PipelineConfigs() map[string]PipelineConfig {
	// Return a copy to avoid mutating globally.
	return copyMap(pipelineConfigs)
}

func copyMap(src map[string]PipelineConfig) map[string]PipelineConfig {
	dst := make(map[string]PipelineConfig)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
