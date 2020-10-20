package release

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagConfig     = "config"
	flagKubeconfig = "kubeconfig"
	flagOutput     = "output"
	flagProvider   = "provider"
	flagReleases   = "releases"
)

type flag struct {
	Config     string
	Kubeconfig string
	Provider   string
	Output     string
	Releases   string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.Config, flagConfig, "g", "", `The path to the file containing API endpoints and tokens for each provider.`)
	cmd.Flags().StringVarP(&f.Kubeconfig, flagKubeconfig, "k", "", `The path to the directory containing the kubeconfig for provider control planes.`)
	cmd.Flags().StringVar(&f.Output, flagOutput, "", `The directory in which to store the release name of the created release.`)
	cmd.Flags().StringVarP(&f.Provider, flagProvider, "p", "", `The provider of the target release.`)
	cmd.Flags().StringVarP(&f.Releases, flagReleases, "s", "", `The path of the releases repo on the local filesystem.`)
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
	if f.Releases == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagReleases)
	}

	return nil
}
