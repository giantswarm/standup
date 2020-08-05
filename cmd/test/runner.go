package test

import (
	"context"
	"io"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v3/pkg/k8srestconfig"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/standup/pkg/gsclient"
	"github.com/giantswarm/standup/pkg/test"
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

			Email:    r.flag.Email,
			Endpoint: r.flag.Endpoint,
			Password: r.flag.Password,
		}

		var err error
		gsClient, err = gsclient.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create REST config for the control plane
	var restConfig *rest.Config
	if r.flag.InCluster == true {
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

	// Read the Release CR from the filesystem
	var release v1alpha1.Release
	releaseYAML, err := ioutil.ReadFile("releases/" + r.flag.Provider + "/v" + r.flag.Release + "/release.yaml")
	if err != nil {
		return microerror.Mask(err)
	}

	// Unmarshal the release
	err = yaml.Unmarshal(releaseYAML, &release)
	if err != nil {
		return microerror.Mask(err)
	}

	// Randomize the name
	release.Name = release.Name+"-"+strconv.Itoa(int(time.Now().Unix()))

	// Create the Release CR
	_, err = k8sClient.G8sClient().ReleaseV1alpha1().Releases().Create(context.Background(), &release, v1.CreateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	// TODO: wait for the release to be ready

	// Create the cluster under test
	clusterID, err := gsClient.CreateCluster(context.Background(), r.flag.Release, r.flag.Provider)
	if err != nil {
		return microerror.Mask(err)
	}

	// Create a keypair for the new tenant cluster
	keyPair, err := gsClient.GetKeypair(context.Background(), clusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	// Create a REST config for the new tenant cluster
	var tenantConfig *rest.Config
	{
		c := k8srestconfig.Config{
			Logger: r.logger,

			Address:   "",
			InCluster: false,
			TLS: k8srestconfig.ConfigTLS{
				CAData:  []byte(keyPair.CertificateAuthorityData),
				CrtData: []byte(keyPair.ClientCertificateData),
				KeyData: []byte(keyPair.ClientKeyData),
			},
		}
		tenantConfig, err = k8srestconfig.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create the test client which runs tests on an existing tenant cluster
	testClient, err := test.New(test.Config{
		Config: tenantConfig,
	})
	if err != nil {
		return microerror.Mask(err)
	}

	// Actually run a test
	err = testClient.RunKubernetesConformance(context.Background())
	if err != nil {
		return microerror.Mask(err)
	}

	// Clean up
	err = gsClient.DeleteCluster(context.Background(), clusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
