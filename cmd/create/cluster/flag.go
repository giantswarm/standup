package cluster

import (
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagConfig     = "config"
	flagKubeconfig = "kubeconfig"
	flagOutput     = "output"
	flagProvider   = "provider"
	flagRelease    = "release"
)

type flag struct {
	Config     string
	Kubeconfig string
	Provider   string
	Output     string
	Release    string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.Config, flagConfig, "g", "", `The path to the file containing API endpoints and tokens for each provider.`)
	cmd.Flags().StringVarP(&f.Kubeconfig, flagKubeconfig, "k", "", `The path to the directory containing the kubeconfigs for provider control planes.`)
	cmd.Flags().StringVar(&f.Output, flagOutput, "", `The directory in which to store the cluster ID, kubeconfig, and provider of the created cluster.`)
	cmd.Flags().StringVarP(&f.Provider, flagProvider, "p", "", `The provider of the target control plane.`)
	cmd.Flags().StringVarP(&f.Release, flagRelease, "r", "", `The semantic version of the release to be tested.`)
}

func (f *flag) Validate() error {
	if f.Config == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagConfig)
	}
	if f.Output == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagOutput)
	}
	if f.Release != "" {
		f.Release = strings.TrimPrefix(f.Release, "v")
		if _, err := semver.NewVersion(f.Release); err != nil {
			return microerror.Maskf(invalidFlagError, "--%s must be a valid semantic version", flagRelease)
		}
		if f.Provider == "" {
			return microerror.Maskf(invalidFlagError, "--%s must be specified when defining an exact release version", flagRelease)
		}
	}

	return nil
}
