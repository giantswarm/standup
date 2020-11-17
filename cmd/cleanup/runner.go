package cleanup

import (
	"context"
	"io"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/standup/pkg/config"
	"github.com/giantswarm/standup/pkg/gsclient"
	"github.com/giantswarm/standup/pkg/key"
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

func (r *runner) run(ctx context.Context, _ *cobra.Command, _ []string) error {
	var providerConfig *config.ProviderConfig
	{
		var err error
		providerConfig, err = config.LoadProviderConfig(r.flag.Config, r.flag.Provider)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create a GS API client for managing tenant clusters
	var gsClient *gsclient.Client
	{
		c := gsclient.Config{
			Logger: r.logger,

			Endpoint: providerConfig.Endpoint,
			Username: providerConfig.Username,
			Password: providerConfig.Password,
			Token:    providerConfig.Token,
		}

		var err error
		gsClient, err = gsclient.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	kubeconfigPath := key.KubeconfigPath(r.flag.Kubeconfig, r.flag.Provider)

	// Create REST config for the control plane
	var restConfig *rest.Config
	{
		var err error
		restConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
			&clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create k8s clients for the control plane
	var k8sClient k8sclient.Interface
	{
		var err error
		k8sClient, err = k8sclient.NewClients(k8sclient.ClientsConfig{
			Logger:     r.logger,
			RestConfig: restConfig,
		})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Get release version of tenant cluster
	var releaseVersion string
	{
		if r.flag.ReleaseID != "" {
			releaseVersion = r.flag.ReleaseID
		} else {
			var err error
			releaseVersion, err = gsClient.GetClusterReleaseVersion(ctx, r.flag.ClusterID)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	r.logger.LogCtx(ctx, "message", "beginning teardown")

	r.logger.LogCtx(ctx, "message", "deleting cluster")
	{
		err := gsClient.DeleteCluster(ctx, r.flag.ClusterID)
		if gsclient.IsClusterNotFoundError(err) {
			r.logger.LogCtx(ctx, "message", "cluster does not exist")
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		}

		// Wait for the cluster to be deleted
		o := func() error {
			clusters, err := gsClient.ListClusters(ctx)
			if err != nil {
				return backoff.Permanent(err)
			}
			for _, cluster := range clusters {
				if cluster.ID == r.flag.ClusterID {
					r.logger.LogCtx(ctx, "message", "waiting for cluster deletion")
					return microerror.Mask(notYetDeletedError)
				}
			}
			return nil
		}
		// Retry basically forever, the tekton task will determine maximum runtime.
		b := backoff.NewMaxRetries(^uint64(0), 20*time.Second)

		err = backoff.Retry(o, b)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	r.logger.LogCtx(ctx, "message", "deleted cluster")

	// Delete the Release CR
	r.logger.LogCtx(ctx, "message", "deleting release CR")
	{
		backgroundDeletion := v1.DeletionPropagation("Background")
		err := k8sClient.G8sClient().ReleaseV1alpha1().Releases().Delete(ctx, releaseVersion, v1.DeleteOptions{
			PropagationPolicy: &backgroundDeletion,
		})
		if err != nil {
			return microerror.Mask(err)
		}

		// Wait for the release to be deleted
		o := func() error {
			_, err := k8sClient.G8sClient().ReleaseV1alpha1().Releases().Get(ctx, releaseVersion, v1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return nil
			} else if err != nil {
				return backoff.Permanent(err)
			}
			r.logger.LogCtx(ctx, "message", "waiting for release deletion")
			return microerror.Mask(notYetDeletedError)
		}
		// Retry basically forever, the tekton task will determine maximum runtime.
		b := backoff.NewMaxRetries(^uint64(0), 20*time.Second)

		err = backoff.Retry(o, b)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	r.logger.LogCtx(ctx, "message", "deleted release CR")
	r.logger.LogCtx(ctx, "message", "teardown complete")

	return nil
}
