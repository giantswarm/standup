package testoperatorrelease

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagConfig       = "config"
	flagKubeconfig   = "kubeconfig"
	flagOperatorPath = "operator-path"
	flagOutput       = "output"
	flagProvider     = "provider"
	flagReleasesPath = "releases-path"
)

type flag struct {
	Config       string
	Kubeconfig   string
	OperatorPath string
	Output       string
	Provider     string
	ReleasesPath string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.Config, flagConfig, "g", "", `The path to the file containing API endpoints and tokens for each provider.`)
	cmd.Flags().StringVarP(&f.Kubeconfig, flagKubeconfig, "k", "", `The path to the directory containing the kubeconfigs for provider control planes.`)
	cmd.Flags().StringVar(&f.OperatorPath, flagOperatorPath, "", `The path of the provider operator repo on the local filesystem.`)
	cmd.Flags().StringVar(&f.Output, flagOutput, "", `The directory in which to store the release name of the created release.`)
	cmd.Flags().StringVar(&f.Provider, flagProvider, "", `The cloud provider to clone the release for.`)
	cmd.Flags().StringVar(&f.ReleasesPath, flagReleasesPath, "", `The path of the releases repo on the local filesystem.`)
}

func (f *flag) Validate() error {
	if f.Config == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagConfig)
	}
	if f.Kubeconfig == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagKubeconfig)
	}
	if f.Output == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagOutput)
	}
	if f.Provider == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagProvider)
	}
	if f.Provider != "azure" && f.Provider != "aws" {
		return microerror.Maskf(invalidFlagError, "The only supported providers are 'azure' and 'aws'")
	}
	if f.OperatorPath == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagOperatorPath)
	}
	if f.ReleasesPath == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagReleasesPath)
	}

	return nil
}
