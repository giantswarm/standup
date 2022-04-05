package cluster

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/organization-operator/api/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

var Scheme = runtime.NewScheme()

func init() {
	rand.Seed(time.Now().Unix())

	schemeBuilder := runtime.SchemeBuilder{
		v1alpha1.AddToScheme,
	}
	err := schemeBuilder.AddToScheme(Scheme)
	if err != nil {
		panic(err)
	}
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
	kubeconfigPath := key.KubeconfigPath(r.flag.Kubeconfig, r.flag.Installation)

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

	ctrl, err := client.New(rest.CopyConfig(restConfig), client.Options{Scheme: Scheme})
	if err != nil {
		return microerror.Mask(err)
	}

	var providerConfig *config.ProviderConfig
	{
		var err error
		providerConfig, err = config.LoadProviderConfig(r.flag.Config, r.flag.Installation)
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

	var organization string
	{
		labelSelector := client.MatchingLabels{
			"giantswarm.io/conformance-testing": "true",
		}
		organizations := &v1alpha1.OrganizationList{}
		err = ctrl.List(ctx, organizations, labelSelector)
		if err != nil {
			return microerror.Mask(err)
		}
		if len(organizations.Items) == 0 {
			return microerror.Mask(notAvailableOrganizationError)
		}

		organization = organizations.Items[rand.Intn(len(organizations.Items))].Name // #nosec
	}

	// Create the cluster under test
	var clusterID string
	r.logger.LogCtx(ctx, "message", fmt.Sprintf("creating cluster using target release %s and organization %s", r.flag.Release, organization))
	{
		var err error
		clusterID, err = gsClient.CreateCluster(ctx, organization, r.flag.Release)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	r.logger.LogCtx(ctx, "message", fmt.Sprintf("created cluster %s", clusterID))

	// Write cluster ID to filesystem
	{
		clusterIDPath := filepath.Join(r.flag.Output, "cluster-id")
		r.logger.LogCtx(ctx, "message", fmt.Sprintf("writing cluster ID to path %s", clusterIDPath))
		err := ioutil.WriteFile(clusterIDPath, []byte(clusterID), 0644) // #nosec
		if err != nil {
			return microerror.Mask(err)
		}
	}

	clusterKubeconfigPath := filepath.Join(r.flag.Output, "kubeconfig")
	r.logger.LogCtx(ctx, "message", fmt.Sprintf("creating and writing kubeconfig for cluster %s to path %s", clusterID, clusterKubeconfigPath))
	{
		o := func() error {
			// Create a keypair and kubeconfig for the new tenant cluster
			err := gsClient.CreateKubeconfig(ctx, clusterID, clusterKubeconfigPath)
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
