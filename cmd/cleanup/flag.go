package cleanup

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

const (
	flagConfig       = "config"
	flagClusterID    = "cluster"
	flagKubeconfig   = "kubeconfig"
	flagInstallation = "installation"
	flagReleaseID    = "release"
)

type flag struct {
	ClusterID    string
	Config       string
	Kubeconfig   string
	Installation string
	ReleaseID    string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.ClusterID, flagClusterID, "c", "", `The ID of the cluster to delete.`)
	cmd.Flags().StringVarP(&f.Config, flagConfig, "g", "", `The path to the file containing API endpoints and tokens for each provider.`)
	cmd.Flags().StringVarP(&f.Kubeconfig, flagKubeconfig, "k", "", `The path to the directory containing the kubeconfigs for provider control planes.`)
	cmd.Flags().StringVarP(&f.Installation, flagInstallation, "i", "", `The target management cluster type to be used ('aws', 'azure', 'kvm', 'gce', 'openstack' or 'aws-china').`)
	cmd.Flags().StringVarP(&f.ReleaseID, flagReleaseID, "r", "", `The release to delete. Defaults to the release of the passed cluster.`)
}

func (f *flag) Validate() error {
	if f.ClusterID == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagClusterID)
	}

	if f.Config == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagConfig)
	}

	if f.Kubeconfig == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagKubeconfig)
	}

	if f.Installation == "" {
		return microerror.Maskf(invalidFlagError, "--%s is required", flagInstallation)
	}

	return nil
}
