package release

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/blang/semver"
	"github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/standup/pkg/git"
)

func (r *runner) generateReleaseFromAzureOperatorRepo(ctx context.Context) (*v1alpha1.Release, string, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "Generating test Release CR using azure-operator mode")
	var release v1alpha1.Release
	provider := "azure"

	// Find latest release in the releases repo.
	r.logger.LogCtx(ctx, "message", "determining latest release")
	var releasePath string
	{
		entries, err := ioutil.ReadDir(filepath.Join(r.flag.Releases, provider))
		if err != nil {
			return nil, "", microerror.Mask(err)
		}

		var latest *semver.Version

		for _, f := range entries {
			if !f.IsDir() || f.Name() == "." || f.Name() == ".." || f.Name() == "archived" {
				continue
			}

			// Try to parse version.
			version, err := semver.Parse(strings.TrimPrefix(f.Name(), "v"))
			if err != nil {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to parse semver version from %q", f.Name()))
				continue
			}

			if latest == nil || version.GT(*latest) {
				latest = &version
			}
		}

		if latest == nil {
			return nil, "", microerror.Maskf(releaseNotFoundError, "Can't find a valid release")
		}

		r.logger.LogCtx(ctx, "message", fmt.Sprintf("latest release is %s", latest.String()))

		releasePath = filepath.Join(r.flag.Releases, provider, fmt.Sprintf("v%s", latest.String()), "release.yaml")
	}

	releaseYAML, err := ioutil.ReadFile(releasePath)
	if err != nil {
		return nil, "", microerror.Mask(err)
	}

	err = yaml.Unmarshal(releaseYAML, &release)
	if err != nil {
		return nil, "", microerror.Mask(err)
	}

	// Get info about the branch of azure operator being tested.
	var headSHA string
	var azureOperatorVersion string
	{
		headSHA, err = git.HeadSHA(r.flag.AzureOperator)
		if err != nil {
			return nil, "", microerror.Mask(err)
		}

		f, err := os.Open(filepath.Join(r.flag.AzureOperator, "pkg/project/project.go"))
		if err != nil {
			return nil, "", microerror.Mask(err)
		}
		defer f.Close()

		// We want to extract the "5.2.1-dev" part (without quotes) from the following line:
		// \tversion            = "5.2.1-dev"
		re := regexp.MustCompile(`^\t*\s*\t*version\s*=\s*"([^"]*)".*$`)
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			matches := re.FindStringSubmatch(scanner.Text())
			if len(matches) == 2 {
				azureOperatorVersion = matches[1]
				break
			}
		}
	}

	// Override the azure operator component.
	for i, c := range release.Spec.Components {
		if c.Name == "azure-operator" {
			release.Spec.Components[i].Version = azureOperatorVersion
			release.Spec.Components[i].Catalog = "control-plane-test-catalog"
			release.Spec.Components[i].Reference = fmt.Sprintf("%s-%s", c.Version, headSHA)
			break
		}
	}

	// TODO update other components based on the requests file.

	return &release, provider, nil
}
