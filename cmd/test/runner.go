package test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

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

func runGit(args []string, dir string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", microerror.Mask(err)
	}
	return string(output), nil
}

// Tekton checks out the current commit in detached HEAD state with --depth=1.
// This means we need to fetch origin/master before we can determine the changed files.
func fetchAndDiff(dir string) (string, error) {
	{
		// Fetch master so we can diff against it
		argsArr := []string{
			"fetch",
			"origin",
			"master",
		}
		_, err := runGit(argsArr, dir)
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	var diff string
	{
		// Determine the files added in this branch not in master
		argsArr := []string{
			"diff",
			"--name-status",   // only show filename and the type of change (A=added, etc.)
			"origin/master",   // diff against the latest master
			"--diff-filter=A", // only show added files
			"HEAD",            // base ref for the diff
		}
		var err error
		diff, err = runGit(argsArr, dir)
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	return diff, nil
}

func findNewRelease(diff string) (releasePath string, provider string, release string, err error) {
	{
		lines := strings.Split(diff, "\n")
		for _, line := range lines {
			if strings.HasSuffix(line, "/release.yaml") {
				fields := strings.Fields(line)
				if len(fields) < 2 {
					err = errors.New(fmt.Sprintf("incorrectly formatted diff: should look like 'A       aws/v13.0.0/release.yaml', found %s", line))
					return
				}
				releasePath = fields[1]
				break
			}
		}
	}

	if releasePath == "" {
		err = errors.New("no new release found in this branch")
		return
	}

	components := strings.Split(releasePath, "/")
	provider = components[0]
	release = components[1]

	return
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

	var release v1alpha1.Release
	var provider string
	var releaseVersion string
	{
		// Use "git diff" to find the release under test
		diff, err := fetchAndDiff(r.flag.Releases)
		if err != nil {
			return microerror.Mask(err)
		}

		// Parse the git diff to get the release file, version, and provider
		var releasePath string
		releasePath, provider, releaseVersion, err = findNewRelease(diff)
		if err != nil {
			return microerror.Mask(err)
		}

		var release v1alpha1.Release
		releaseYAML, err := ioutil.ReadFile(releasePath)
		if err != nil {
			return microerror.Mask(err)
		}

		err = yaml.Unmarshal(releaseYAML, &release)
		if err != nil {
			return microerror.Mask(err)
		}

		// Randomize the name to avoid duplicate names
		release.Name = release.Name + "-" + strconv.Itoa(int(time.Now().Unix()))
		// Label for future garbage collection
		release.Labels["giantswarm.io/testing"] = "true"
	}

	// Create the Release CR
	_, err = k8sClient.G8sClient().ReleaseV1alpha1().Releases().Create(context.Background(), &release, v1.CreateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	// Wait for the created release to be ready
	{
		o := func() error {
			r, err := k8sClient.G8sClient().ReleaseV1alpha1().Releases().Get(context.Background(), release.Name, v1.GetOptions{})
			if err != nil {
				return backoff.Permanent(err)
			}
			if !r.Status.Ready {
				return errors.New("not ready")
			}

			return nil
		}

		b := backoff.NewMaxRetries(10, 20*time.Second)

		err = backoff.Retry(o, b)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create the cluster under test
	clusterID, err := gsClient.CreateCluster(context.Background(), releaseVersion, provider)
	if err != nil {
		return microerror.Mask(err)
	}

	// TODO: Wait + backoff instead of just sleeping
	// PKI backend needs some time after cluster creation
	time.Sleep(5 * time.Second)

	// Create a keypair for the new tenant cluster
	kubeconfig, err := gsClient.GetKubeconfig(context.Background(), clusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	// TODO: Store me somewhere
	fmt.Println(len(kubeconfig))

	// Clean up
	err = gsClient.DeleteCluster(context.Background(), clusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	// TODO: Delete release

	return nil
}
