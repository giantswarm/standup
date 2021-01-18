package release

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagAzureOperator = "azure-operator"
	flagConfig        = "config"
	flagKubeconfig    = "kubeconfig"
	flagMode          = "mode"
	flagOutput        = "output"
	flagReleases      = "releases"

	modeReleases      = "releases"
	modeAzureOperator = "azure-operator"
)

type flag struct {
	AzureOperator string
	Config        string
	Kubeconfig    string
	Mode          string
	Output        string
	Releases      string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.AzureOperator, flagAzureOperator, "", "", `The path to the file containing API endpoints and tokens for each provider.`)
	cmd.Flags().StringVarP(&f.Config, flagConfig, "g", "", `The path to the file containing API endpoints and tokens for each provider.`)
	cmd.Flags().StringVarP(&f.Kubeconfig, flagKubeconfig, "k", "", `The path to the directory containing the kubeconfigs for provider control planes.`)
	cmd.Flags().StringVar(&f.Mode, flagMode, "releases", `The mode to be used to create the release: either 'releases' or 'azure-operator'.`)
	cmd.Flags().StringVar(&f.Output, flagOutput, "", `The directory in which to store the release name of the created release.`)
	cmd.Flags().StringVarP(&f.Releases, flagReleases, "s", "", `The path of the releases repo on the local filesystem.`)
}

func (f *flag) Validate() error {
	if f.Mode == modeAzureOperator && f.AzureOperator == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required when --mode is set to %s", flagAzureOperator, modeAzureOperator)
	}
	if f.Config == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagConfig)
	}
	if f.Kubeconfig == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagKubeconfig)
	}
	if f.Mode != modeReleases && f.Mode != modeAzureOperator {
		return microerror.Maskf(invalidFlagError, "--%s can either be %q or %q. %q was provided", flagMode, modeReleases, modeAzureOperator, f.Mode)
	}
	if f.Output == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagOutput)
	}
	if f.Releases == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagReleases)
	}

	return nil
}
