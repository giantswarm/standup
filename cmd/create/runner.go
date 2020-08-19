package create

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"path/filepath"
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
	"k8s.io/apimachinery/pkg/types"
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
			"--no-renames",    // disable rename detection so we always find new releases
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

func findNewRelease(diff string) (releasePath string, provider string, err error) {
	{
		lines := strings.Split(diff, "\n")
		for _, line := range lines {
			fmt.Println("checking line: ", line)
			if strings.HasSuffix(line, "/release.yaml") {
				fields := strings.Fields(line)
				if len(fields) < 2 {
					err = fmt.Errorf("incorrectly formatted diff: should look like 'A       aws/v13.0.0/release.yaml', found %s", line)
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

	return
}

func (r *runner) run(ctx context.Context, _ *cobra.Command, _ []string) error {
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
	r.logger.LogCtx(ctx, "message", "determining release to test")
	{
		var releasePath string
		{
			if r.flag.Release != "" {
				// Read the Release CR with the given version from the filesystem
				releasePath = filepath.Join(r.flag.Releases, r.flag.Provider, "/v"+r.flag.Release, "/release.yaml")
				provider = r.flag.Provider
			} else {
				// Use "git diff" to find the release under test
				diff, err := fetchAndDiff(r.flag.Releases)
				if err != nil {
					return microerror.Mask(err)
				}
				r.logger.LogCtx(ctx, "message", "calculated diff", "diff", diff)

				// Parse the git diff to get the release file, version, and provider
				releasePath, provider, err = findNewRelease(diff)
				if err != nil {
					return microerror.Mask(err)
				}
				releasePath = filepath.Join(r.flag.Releases, releasePath)
			}
		}

		r.logger.LogCtx(ctx, "message", "determined target release to test is "+releasePath)

		releaseYAML, err := ioutil.ReadFile(releasePath)
		if err != nil {
			return microerror.Mask(err)
		}

		err = yaml.Unmarshal(releaseYAML, &release)
		if err != nil {
			return microerror.Mask(err)
		}

		// Randomize the name to avoid duplicate names
		originalName := release.Name
		release.Name = release.Name + "-" + strconv.Itoa(int(time.Now().Unix()))
		releaseVersion = strings.TrimPrefix(release.Name, "v")
		r.logger.LogCtx(ctx, "message", fmt.Sprintf("testing release %s for %s as %s", strings.TrimPrefix(originalName, "v"), provider, releaseVersion))

		// Label for future garbage collection
		if release.Labels == nil {
			release.Labels = map[string]string{}
		}
		release.Labels["giantswarm.io/testing"] = "true"

		if r.flag.OwnerName != "" {
			blockOwnerDeletion := true
			release.OwnerReferences = append(release.OwnerReferences, v1.OwnerReference{
				APIVersion:         "prow.k8s.io/v1",
				BlockOwnerDeletion: &blockOwnerDeletion,
				Kind:               "ProwJob",
				Name:               r.flag.OwnerName,
				UID:                types.UID(r.flag.OwnerUID),
			})
		}
	}

	// Create the Release CR
	r.logger.LogCtx(ctx, "message", "creating release CR")
	_, err = k8sClient.G8sClient().ReleaseV1alpha1().Releases().Create(ctx, &release, v1.CreateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "message", "created release CR")

	// Wait for the created release to be ready
	r.logger.LogCtx(ctx, "message", "waiting for release to be ready")
	{
		o := func() error {
			toCheck, err := k8sClient.G8sClient().ReleaseV1alpha1().Releases().Get(ctx, release.Name, v1.GetOptions{})
			if err != nil {
				return backoff.Permanent(err)
			}
			if !toCheck.Status.Ready {
				r.logger.LogCtx(ctx, "message", "release is not ready yet")
				return errors.New("not ready")
			}

			return nil
		}

		b := backoff.NewMaxRetries(30, 20*time.Second)

		err = backoff.Retry(o, b)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	r.logger.LogCtx(ctx, "message", "release is ready")

	// Create the cluster under test
	r.logger.LogCtx(ctx, "message", "creating cluster using target release")
	clusterID, err := gsClient.CreateCluster(ctx, releaseVersion)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "message", fmt.Sprintf("created cluster %s", clusterID))

	var kubeconfig string
	r.logger.LogCtx(ctx, "message", "creating kubeconfig for cluster")
	{
		o := func() error {
			// Create a keypair for the new tenant cluster
			kubeconfig, err = gsClient.GetKubeconfig(ctx, clusterID)
			if err != nil {
				// TODO: check to see if it's a permanent error or the kubeconfig just isn't ready yet
				r.logger.LogCtx(ctx, "message", "error creating kubeconfig", "error", err)
				return microerror.Mask(err)
			}
			return nil
		}

		b := backoff.NewMaxRetries(10, 20*time.Second)

		err = backoff.Retry(o, b)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	r.logger.LogCtx(ctx, "message", fmt.Sprintf("created kubeconfig with length %d", len(kubeconfig)))

	r.logger.LogCtx(ctx, "message", fmt.Sprintf("writing kubeconfig to path %s", r.flag.OutputKubeconfig))
	err = ioutil.WriteFile(r.flag.OutputKubeconfig, []byte(kubeconfig), 0644)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "message", fmt.Sprintf("writing cluster ID to path %s", r.flag.OutputClusterID))
	err = ioutil.WriteFile(r.flag.OutputClusterID, []byte(clusterID), 0644)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "message", "setup complete")

	return nil
}
