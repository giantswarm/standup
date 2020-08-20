package create

import (
	"fmt"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	"github.com/giantswarm/standup/pkg/gsclient"
)

const (
	defaultOutputClusterIDPath  = "/workspace/cluster/cluster-id"
	defaultOutputKubeconfigPath = "/workspace/cluster/kubeconfig"

	flagKubeconfig       = "kubeconfig"
	flagEndpoint         = "endpoint"
	flagInCluster        = "in-cluster"
	flagOutputClusterID  = "output-cluster-id"
	flagOutputKubeconfig = "output-kubeconfig"
	flagProvider         = "provider"
	flagRelease          = "release"
	flagReleases         = "releases"
	flagToken            = "token"
)

type flag struct {
	Kubeconfig       string
	Endpoint         string
	InCluster        bool
	Provider         string
	OutputClusterID  string
	OutputKubeconfig string
	Release          string
	Releases         string
	Token            string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.Kubeconfig, flagKubeconfig, "k", "", `The path to the kubeconfig for the control plane.`)
	cmd.Flags().StringVarP(&f.Endpoint, flagEndpoint, "n", "", `The endpoint of the target control plane's API.`)
	cmd.Flags().BoolVarP(&f.InCluster, flagInCluster, "i", false, `True if this program is running in a Kubernetes cluster and should communicate with the API via the injected service account token.`)
	cmd.Flags().StringVar(&f.OutputClusterID, flagOutputClusterID, "", fmt.Sprintf(`The path of the file in which to store the cluster ID of the created cluster.`))
	cmd.Flags().StringVar(&f.OutputKubeconfig, flagOutputKubeconfig, "", fmt.Sprintf(`The path of the file in which to store the kubeconfig of the created cluster.`))
	cmd.Flags().StringVarP(&f.Provider, flagProvider, "p", "", fmt.Sprintf(`The provider of the target release. Possible values: <%s>`, strings.Join(gsclient.AllProviders(), "|")))
	cmd.Flags().StringVarP(&f.Release, flagRelease, "r", "", fmt.Sprintf(`The semantic version of the release to be tested.`))
	cmd.Flags().StringVarP(&f.Releases, flagReleases, "s", "", fmt.Sprintf(`The path of the releases repo on the local filesystem.`))
	cmd.Flags().StringVarP(&f.Token, flagToken, "t", "", `The token used to authenticate with the GS API.`)
}

func (f *flag) Validate() error {
	if f.Kubeconfig == "" && !f.InCluster || f.Kubeconfig != "" && f.InCluster {
		return microerror.Maskf(invalidFlagError, "--%s and --%s are mutually exclusive", flagKubeconfig, flagInCluster)
	}
	if f.OutputClusterID == "" {
		f.OutputClusterID = defaultOutputClusterIDPath
	}
	if f.OutputKubeconfig == "" {
		f.OutputKubeconfig = defaultOutputKubeconfigPath
	}
	if f.Releases == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagReleases)
	}
	if f.Release != "" {
		// Remove leading v from release version
		f.Release = strings.TrimPrefix(f.Release, "v")
	}
	if f.Release != "" && !gsclient.IsValidRelease(f.Release) {
		return microerror.Maskf(invalidFlagError, "--%s must be a valid semantic version", flagRelease)
	}
	if f.Release != "" && f.Provider == "" {
		return microerror.Maskf(invalidFlagError, "must specify a valid provider when setting an exact release version")
	}

	return nil
}
