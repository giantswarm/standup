package cleanup

import (
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	"github.com/giantswarm/standup/pkg/gsclient"
)

const (
	flagClusterID  = "cluster"
	flagKubeconfig = "kubeconfig"
	flagEndpoint   = "endpoint"
	flagInCluster  = "in-cluster"
	flagRelease    = "release"
	flagToken      = "token"
)

type flag struct {
	ClusterID  string
	Kubeconfig string
	Endpoint   string
	InCluster  bool
	Release    string
	Token      string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.ClusterID, flagClusterID, "c", "", `The ID of the cluster to delete.`)
	cmd.Flags().StringVarP(&f.Kubeconfig, flagKubeconfig, "k", "", `The path to the kubeconfig for the control plane.`)
	cmd.Flags().StringVarP(&f.Endpoint, flagEndpoint, "n", "", `The endpoint of the target control plane's API.`)
	cmd.Flags().BoolVarP(&f.InCluster, flagInCluster, "i", false, `True if this program is running in a Kubernetes cluster and should communicate with the API via the injected service account token.`)
	cmd.Flags().StringVarP(&f.Release, flagRelease, "r", "", fmt.Sprintf(`The semantic version of the release to be tested.`))
	cmd.Flags().StringVarP(&f.Token, flagToken, "t", "", `The token used to authenticate with the GS API.`)
}

func (f *flag) Validate() error {
	if f.Kubeconfig == "" && !f.InCluster || f.Kubeconfig != "" && f.InCluster {
		return microerror.Maskf(invalidFlagError, "--%s and --%s are mutually exclusive", flagKubeconfig, flagInCluster)
	}
	if !gsclient.IsValidRelease(f.Release) {
		return microerror.Maskf(invalidFlagError, "--%s must be a valid semantic version", flagRelease)
	}

	return nil
}
