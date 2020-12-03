package cluster

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/standup/pkg/config"
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

	// Create the cluster under test
	var clusterID string
	r.logger.LogCtx(ctx, "message", "creating cluster using target release")
	{
		var err error
		clusterID, err = gsClient.CreateCluster(ctx, r.flag.Release)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	r.logger.LogCtx(ctx, "message", fmt.Sprintf("created cluster %s", clusterID))

	// Write cluster ID to filesystem
	{
		clusterIDPath := filepath.Join(r.flag.Output, "cluster-id")
		r.logger.LogCtx(ctx, "message", fmt.Sprintf("writing cluster ID to path %s", clusterIDPath))
		err := ioutil.WriteFile(clusterIDPath, []byte(clusterID), 0644)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	kubeconfigPath := filepath.Join(r.flag.Output, "kubeconfig")
	r.logger.LogCtx(ctx, "message", fmt.Sprintf("creating and writing kubeconfig for cluster %s to path %s", clusterID, kubeconfigPath))
	{
		o := func() error {
			// Create a keypair and kubeconfig for the new tenant cluster
			err := gsClient.CreateKubeconfig(ctx, clusterID, kubeconfigPath)
			if err != nil {
				r.logger.LogCtx(ctx, "message", "error creating kubeconfig", "error", err)
				return microerror.Mask(err)
			}
			return nil
		}
		// Retry basically forever, the tekton task will determine maximum runtime.
		b := backoff.NewMaxRetries(^uint64(0), 20*time.Second)

		err := backoff.Retry(o, b)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "message", "setup complete")

	return nil
}
