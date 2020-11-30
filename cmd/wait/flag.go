package wait

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagKubeconfig = "kubeconfig"
	flagProvider   = "provider"
)

type flag struct {
	Kubeconfig string
	Provider   string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.Kubeconfig, flagKubeconfig, "k", "", `The path to the kubeconfig for the tenant cluster.`)
	cmd.Flags().StringVarP(&f.Provider, flagProvider, "p", "", `The provider of the target control plane.`)
}

func (f *flag) Validate() error {
	if f.Kubeconfig == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagKubeconfig)
	}
	if f.Provider == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagProvider)
	}

	return nil
}
