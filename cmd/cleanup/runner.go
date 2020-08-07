package cleanup

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	// v1 "k8s.io/client-go/1.5/pkg/api/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
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
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(context.Background(), "message", "beginning teardown")

	r.logger.LogCtx(context.Background(), "message", "deleting cluster")
	err = gsClient.DeleteCluster(context.Background(), r.flag.ClusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	// Wait for the cluster to be deleted
	{
		o := func() error {
			clusters, err := gsClient.ListClusters(context.Background())
			if err != nil {
				return backoff.Permanent(err)
			}
			for _, cluster := range clusters {
				if cluster.ID == r.flag.ClusterID {
					r.logger.LogCtx(context.Background(), "message", "waiting for cluster deletion")
					return errors.New("waiting for cluster deletion")
				}
			}
			return nil
		}

		b := backoff.NewMaxRetries(100, 20*time.Second)

		err = backoff.Retry(o, b)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	r.logger.LogCtx(context.Background(), "message", "deleted cluster")

	// Delete the Release CR
	r.logger.LogCtx(context.Background(), "message", "deleting release CR")
	backgroundDeletion := v1.DeletionPropagation("Background")
	err = k8sClient.G8sClient().ReleaseV1alpha1().Releases().Delete(context.Background(), releaseVersion, v1.DeleteOptions{
		PropagationPolicy: &backgroundDeletion,
	})
	if err != nil {
		return microerror.Mask(err)
	}

	// Wait for the release to be deleted
	{
		o := func() error {
			_, err := k8sClient.G8sClient().ReleaseV1alpha1().Releases().Get(context.Background(), releaseVersion, v1.GetOptions{})
			if errors2.IsNotFound(err) {
				return nil
			} else if err != nil {
				return backoff.Permanent(err)
			}
			r.logger.LogCtx(context.Background(), "message", "waiting for release deletion")
			return errors.New("waiting for release deletion")
		}

		b := backoff.NewMaxRetries(10, 20*time.Second)

		err = backoff.Retry(o, b)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	r.logger.LogCtx(context.Background(), "message", "deleted release CR")
	r.logger.LogCtx(context.Background(), "message", "teardown complete")

	return nil
}
