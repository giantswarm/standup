package release

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/standup/pkg/git"
)

func (r *runner) generateReleaseFromReleasesRepo(ctx context.Context) (*v1alpha1.Release, string, error) {
	var release v1alpha1.Release
	var provider string
	r.logger.LogCtx(ctx, "message", "determining release to test")
	{
		var releasePath string
		{
			// Tekton checks out the current commit in detached HEAD state with --depth=1.
			// This means we need to fetch origin/master before we can determine the changed files.
			err := git.Fetch(r.flag.Releases)
			if err != nil {
				return nil, "", microerror.Mask(err)
			}

			mergeBase, err := git.MergeBase(r.flag.Releases)
			if err != nil {
				return nil, "", microerror.Mask(err)
			}

			// Use "git diff" to find the release under test
			diff, err := git.Diff(r.flag.Releases, mergeBase)
			if err != nil {
				return nil, "", microerror.Mask(err)
			}

			// Parse the git diff to get the release file, version, and provider
			releasePath, provider, err = findReleaseInDiff(diff)
			if err != nil {
				return nil, "", microerror.Mask(err)
			}
			releasePath = filepath.Join(r.flag.Releases, releasePath)
		}

		r.logger.LogCtx(ctx, "message", "determined target release to test is "+releasePath)

		releaseYAML, err := ioutil.ReadFile(releasePath)
		if err != nil {
			return nil, "", microerror.Mask(err)
		}

		err = yaml.Unmarshal(releaseYAML, &release)
		if err != nil {
			return nil, "", microerror.Mask(err)
		}
	}

	return &release, provider, nil
}
