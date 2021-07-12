package release

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/standup/pkg/git"
	"github.com/giantswarm/standup/pkg/key"
)

// Following pattern for release name has been taken from CRD validation:
// https://github.com/giantswarm/apiextensions/blob/master/config/crd/patches/v1/release.giantswarm.io_releases/patch.yaml
var releaseNamePattern = regexp.MustCompile(`^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[\.0-9a-zA-Z]*)?$`)

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

func findReleaseInDiff(diff string) (releasePath string, provider string, err error) {
	{
		lines := strings.Split(diff, "\n")
		for _, line := range lines {
			if strings.HasSuffix(line, "/release.yaml") {
				fields := strings.Fields(line)
				if len(fields) < 2 {
					err = microerror.Maskf(releaseNotFoundError, "incorrectly formatted diff: should look like 'A  aws/v13.0.0/release.yaml', found %s", line)
					return
				}
				releasePath = fields[1]
				break
			}
		}
	}

	if releasePath == "" {
		err = microerror.Maskf(releaseNotFoundError, "no new release found in diff between this branch and master")
		return
	}

	components := strings.Split(releasePath, "/")
	provider = components[0]

	return
}

func (r *runner) run(ctx context.Context, _ *cobra.Command, _ []string) error {
	var release v1alpha1.Release
	var provider string
	var installation string
	var releaseVersion string
	r.logger.LogCtx(ctx, "message", "determining release to test")
	{
		var releasePath string
		{
			// Tekton checks out the current commit in detached HEAD state with --depth=1.
			// This means we need to fetch origin/master before we can determine the changed files.
			err := git.Fetch(r.flag.Releases)
			if err != nil {
				return microerror.Mask(err)
			}

			mergeBase, err := git.MergeBase(r.flag.Releases)
			if err != nil {
				return microerror.Mask(err)
			}

			// Use "git diff" to find the release under test
			diff, err := git.Diff(r.flag.Releases, mergeBase)
			if err != nil {
				return microerror.Mask(err)
			}

			// Parse the git diff to get the release file, version, and provider
			releasePath, provider, err = findReleaseInDiff(diff)
			if err != nil {
				return microerror.Mask(err)
			}
			releasePath = filepath.Join(r.flag.Releases, releasePath)
		}

		r.logger.LogCtx(ctx, "message", "determined target release to test is "+releasePath)

		// If this pipeline overrides the target installation, set it. Otherwise use the provider as the target name.
		if i := key.GetInstallationForPipeline(r.flag.Pipeline); i != "" {
			installation = i
		} else {
			installation = provider
		}

		r.logger.LogCtx(ctx, "message", "determined target installation is "+installation)

		releaseYAML, err := ioutil.ReadFile(releasePath)
		if err != nil {
			return microerror.Mask(err)
		}

		err = yaml.Unmarshal(releaseYAML, &release)
		if err != nil {
			return microerror.Mask(err)
		}

		// Randomize the name to avoid duplicate names.
		originalName := release.Name
		release.Name = generateReleaseName(release.Name)
		releaseVersion = strings.TrimPrefix(release.Name, "v")
		r.logger.LogCtx(ctx, "message", fmt.Sprintf("testing release %s for %s as %s", strings.TrimPrefix(originalName, "v"), provider, releaseVersion))

		// Label for future garbage collection
		if release.Labels == nil {
			release.Labels = map[string]string{}
		}
		release.Labels["giantswarm.io/testing"] = "true"
	}

	kubeconfigPath := key.KubeconfigPath(r.flag.Kubeconfig, installation)

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

	// Write the provider name to the filesystem
	{
		providerPath := filepath.Join(r.flag.Output, "provider")
		r.logger.LogCtx(ctx, "message", fmt.Sprintf("writing provider (%s) to path %s", provider, providerPath))
		err := ioutil.WriteFile(providerPath, []byte(provider), 0644) //#nosec
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Write the target installation name to the filesystem
	{
		installationPath := filepath.Join(r.flag.Output, "installation")
		r.logger.LogCtx(ctx, "message", fmt.Sprintf("writing target installation (%s) to path %s", installation, installationPath))
		err := ioutil.WriteFile(installationPath, []byte(installation), 0644) //#nosec
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create the Release CR
	r.logger.LogCtx(ctx, "message", "creating release CR")
	{
		_, err := k8sClient.G8sClient().ReleaseV1alpha1().Releases().Create(ctx, &release, v1.CreateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}
	r.logger.LogCtx(ctx, "message", "created release CR")

	// Write release ID to filesystem
	{
		releaseIDPath := filepath.Join(r.flag.Output, "release-id")
		r.logger.LogCtx(ctx, "message", fmt.Sprintf("writing release ID to path %s", releaseIDPath))
		err := ioutil.WriteFile(releaseIDPath, []byte(release.Name), 0644) //#nosec
		if err != nil {
			return microerror.Mask(err)
		}
	}

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
				return microerror.Mask(releaseNotReadyError)
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
	r.logger.LogCtx(ctx, "message", "release is ready")

	return nil
}

func generateReleaseName(name string) string {
	testSuffix := "-" + strconv.Itoa(int(time.Now().Unix()))

	m := releaseNamePattern.FindStringSubmatch(name)
	if m == nil || m[4] == "" {
		return name + testSuffix
	}

	return strings.Replace(name, m[4], testSuffix, 1)
}
