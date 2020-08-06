package test

import (
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagKubeconfig = "kubeconfig"
	flagEndpoint   = "endpoint"
	flagInCluster  = "in-cluster"
	flagReleases   = "releases"
	flagToken      = "token"
)

type flag struct {
	Kubeconfig string
	Endpoint   string
	InCluster  bool
	Releases   string
	Token      string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.Kubeconfig, flagKubeconfig, "k", "", `The path to the kubeconfig for the control plane.`)
	cmd.Flags().StringVarP(&f.Endpoint, flagEndpoint, "n", "", `The endpoint of the target control plane's API.`)
	cmd.Flags().BoolVarP(&f.InCluster, flagInCluster, "i", false, `True if this program is running in a Kubernetes cluster and should communicate with the API via the injected service account token.`)
	cmd.Flags().StringVarP(&f.Releases, flagReleases, "r", "", fmt.Sprintf(`The path of the releases repo on the local filesystem.`))
	cmd.Flags().StringVarP(&f.Token, flagToken, "t", "", `The token used to authenticate with the GS API.`)
}

func (f *flag) Validate() error {
	if f.Kubeconfig == "" && !f.InCluster || f.Kubeconfig != "" && f.InCluster {
		return microerror.Maskf(invalidFlagError, "--%s and --%s are mutually exclusive", flagKubeconfig, flagInCluster)
	}
	if f.Releases == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagReleases)
	}

	return nil
}
