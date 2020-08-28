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

	flagConfig           = "config"
	flagKubeconfig       = "kubeconfig"
	flagOutputClusterID  = "output-cluster-id"
	flagOutputKubeconfig = "output-kubeconfig"
	flagProvider         = "provider"
	flagRelease          = "release"
	flagReleases         = "releases"
)

type flag struct {
	Config           string
	Kubeconfig       string
	Provider         string
	OutputClusterID  string
	OutputKubeconfig string
	Release          string
	Releases         string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.Config, flagConfig, "c", "", `The path to the file containing API endpoints and tokens for each provider.`)
	cmd.Flags().StringVarP(&f.Kubeconfig, flagKubeconfig, "k", "", `The path to the directory containing the kubeconfig for provider control planes.`)
	cmd.Flags().StringVar(&f.OutputClusterID, flagOutputClusterID, "", fmt.Sprintf(`The path of the file in which to store the cluster ID of the created cluster.`))
	cmd.Flags().StringVar(&f.OutputKubeconfig, flagOutputKubeconfig, "", fmt.Sprintf(`The path of the file in which to store the kubeconfig of the created cluster.`))
	cmd.Flags().StringVarP(&f.Provider, flagProvider, "p", "", `The provider of the target release.`)
	cmd.Flags().StringVarP(&f.Release, flagRelease, "r", "", fmt.Sprintf(`The semantic version of the release to be tested.`))
	cmd.Flags().StringVarP(&f.Releases, flagReleases, "s", "", fmt.Sprintf(`The path of the releases repo on the local filesystem.`))
}

func (f *flag) Validate() error {
	if f.Config == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagConfig)
	}
	if f.Kubeconfig == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagKubeconfig)
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
