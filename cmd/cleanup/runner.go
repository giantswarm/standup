package cleanup

import (
	"context"
	"io"

	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/standup/pkg/gsclient"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	// Create a GS API client for managing tenant clusters
	var gsClient *gsclient.Client
	{
		c := gsclient.Config{
			Logger: r.logger,

			Endpoint: r.flag.Endpoint,
			Token:    r.flag.Token,
		}

		var err error
		gsClient, err = gsclient.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// TODO: Will be used for cleaning up Release from CP
	// Create REST config for the control plane
	var restConfig *rest.Config
	if r.flag.InCluster {
		var err error
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create clients for the control plane
	k8sClient, err := k8sclient.NewClients(k8sclient.ClientsConfig{
		Logger:         r.logger,
		KubeConfigPath: r.flag.Kubeconfig,
		RestConfig:     restConfig,
	})
	if err != nil {
		return microerror.Mask(err)
	}

	// Clean up

	// Get release version of tenant cluster
	releaseVersion, err := gsClient.GetClusterReleaseVersion(context.Background(), r.flag.ClusterID)

	// Delete tenant cluster
	err = gsClient.DeleteCluster(context.Background(), r.flag.ClusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	// Delete the release if we know which one to delete
	if releaseVersion != "" {
		err = k8sClient.G8sClient().ReleaseV1alpha1().Releases().Delete(context.Background(), releaseVersion, v1.DeleteOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
