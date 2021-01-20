package testoperatorrelease

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
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

func (r *runner) run(ctx context.Context, _ *cobra.Command, _ []string) error {
	var err error
	provider := r.flag.Provider
	release, err := r.findLatestRelease(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	// Update release apps and components based on the "requests.yaml" file.
	err = r.updateFromRequests(ctx, release)
	if err != nil {
		return microerror.Mask(err)
	}

	// Update release with current version of provider operator and required dependencies.
	err = r.updateRelease(ctx, release)
	if err != nil {
		return microerror.Mask(err)
	}

	// Randomize the name to avoid duplicate names.
	{
		originalName := release.Name
		release.Name = generateReleaseName(release.Name)
		releaseVersion := strings.TrimPrefix(release.Name, "v")
		r.logger.LogCtx(ctx, "message", fmt.Sprintf("testing release %s for %s as %s", strings.TrimPrefix(originalName, "v"), provider, releaseVersion))
	}

	// Label for future garbage collection
	{
		if release.Labels == nil {
			release.Labels = map[string]string{}
		}
		release.Labels["giantswarm.io/testing"] = "true"
	}

	fmt.Println("DONE")

	// Create release CR.
	//err = r.createRelease(ctx, release)
	//if err != nil {
	//	return microerror.Mask(err)
	//}

	return nil
}

func (r *runner) createRelease(ctx context.Context, release *v1alpha1.Release) error {
	// Create release in the management cluster.
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

	// Write provider to filesystem
	{
		providerPath := filepath.Join(r.flag.Output, "provider")
		r.logger.LogCtx(ctx, "message", fmt.Sprintf("writing provider to path %s", providerPath))
		err := ioutil.WriteFile(providerPath, []byte(r.flag.Provider), 0644)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Create the Release CR
	r.logger.LogCtx(ctx, "message", "creating release CR")
	{
		_, err := k8sClient.G8sClient().ReleaseV1alpha1().Releases().Create(ctx, release, v1.CreateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}
	r.logger.LogCtx(ctx, "message", "created release CR")

	// Write release ID to filesystem
	{
		releaseIDPath := filepath.Join(r.flag.Output, "release-id")
		r.logger.LogCtx(ctx, "message", fmt.Sprintf("writing release ID to path %s", releaseIDPath))
		err := ioutil.WriteFile(releaseIDPath, []byte(release.Name), 0644)
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

func (r *runner) findLatestRelease(ctx context.Context) (*v1alpha1.Release, error) {
	var release v1alpha1.Release
	var releasePath string
	provider := r.flag.Provider
	r.logger.LogCtx(ctx, "message", fmt.Sprintf("determining latest release for provider %s", provider))
	{
		entries, err := ioutil.ReadDir(filepath.Join(r.flag.ReleasesPath, provider))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		var latest *semver.Version

		for _, f := range entries {
			if !f.IsDir() || f.Name() == "." || f.Name() == ".." || f.Name() == "archived" {
				continue
			}

			// Try to parse version.
			version, err := semver.NewVersion(strings.TrimPrefix(f.Name(), "v"))
			if err != nil {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to parse semver version from %q", f.Name()))
				continue
			}

			if latest == nil || version.GreaterThan(latest) {
				latest = version
			}
		}

		if latest == nil {
			return nil, microerror.Maskf(releaseNotFoundError, "Can't find a valid release")
		}

		r.logger.LogCtx(ctx, "message", fmt.Sprintf("latest %s release is %s", provider, latest.String()))

		releasePath = filepath.Join(r.flag.ReleasesPath, provider, fmt.Sprintf("v%s", latest.String()), "release.yaml")
	}

	releaseYAML, err := ioutil.ReadFile(releasePath)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	err = yaml.Unmarshal(releaseYAML, &release)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &release, nil
}

func generateReleaseName(name string) string {
	testSuffix := "-" + strconv.Itoa(int(time.Now().Unix()))

	m := releaseNamePattern.FindStringSubmatch(name)
	if m == nil || m[4] == "" {
		return name + testSuffix
	}

	return strings.Replace(name, m[4], testSuffix, 1)
}

func (r *runner) updateRelease(ctx context.Context, release *v1alpha1.Release) error {
	var err error
	// Get info about the branch of provider operator being tested.
	var headSHA string
	var providerOperatorVersion string
	{
		headSHA, err = git.HeadSHA(r.flag.OperatorPath)
		if err != nil {
			return microerror.Mask(err)
		}

		f, err := os.Open(filepath.Join(r.flag.OperatorPath, "pkg/project/project.go"))
		if err != nil {
			return microerror.Mask(err)
		}
		defer f.Close()

		// We want to extract the "5.2.1-dev" part (without quotes) from the following line:
		// \tversion            = "5.2.1-dev"
		re := regexp.MustCompile(`^\t*\s*\t*version\s*=\s*"([^"]*)".*$`)
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			matches := re.FindStringSubmatch(scanner.Text())
			if len(matches) == 2 {
				providerOperatorVersion = matches[1]
				break
			}
		}
	}

	// Override the provider operator component.
	for i, c := range release.Spec.Components {
		if c.Name == fmt.Sprintf("%s-operator", r.flag.Provider) {
			release.Spec.Components[i].Version = providerOperatorVersion
			release.Spec.Components[i].Catalog = "control-plane-test-catalog"
			release.Spec.Components[i].Reference = fmt.Sprintf("%s-%s", c.Version, headSHA)
			break
		}
	}

	// Override date.
	release.Spec.Date = &v1.Time{
		Time: time.Now(),
	}

	// Mark as WIP.
	release.Spec.State = v1alpha1.StateWIP

	return nil
}
