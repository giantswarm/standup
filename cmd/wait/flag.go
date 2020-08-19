package wait

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagKubeconfig = "kubeconfig"
)

type flag struct {
	Kubeconfig string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.Kubeconfig, flagKubeconfig, "k", "", `The path to the kubeconfig for the control plane.`)
}

func (f *flag) Validate() error {
	if f.Kubeconfig == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagKubeconfig)
	}

	return nil
}
