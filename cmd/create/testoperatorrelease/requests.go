package testoperatorrelease

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
)

type apprequest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Issue   string `json:"issue"`
}

type request struct {
	Name     string       `json:"name"`
	Requests []apprequest `json:"requests"`
}

type requests struct {
	Releases []request `json:"releases"`
}

func (r *runner) updateFromRequests(ctx context.Context, release *v1alpha1.Release) error {
	reqs := requests{}
	// Parse requests from the requests.yaml file.
	{
		requestsPath := filepath.Join(r.flag.ReleasesPath, r.flag.Provider, "requests.yaml")

		releaseYAML, err := os.ReadFile(requestsPath)
		if err != nil {
			return microerror.Mask(err)
		}

		err = yaml.Unmarshal(releaseYAML, &reqs)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	targetVersion, err := semver.NewVersion(strings.TrimPrefix(release.Name, "v"))
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Looking for requests for version %v", targetVersion.String())

	apps, err := r.mergeRequirements(ctx, reqs, *targetVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	for appname, minVersionStr := range apps {
		// Search app or component in the release CR.
		found := false
		for i, comp := range release.Spec.Components {
			if comp.Name == appname {
				found = true
				currentVersion, err := semver.NewVersion(comp.Version)
				if err != nil {
					return microerror.Mask(err)
				}

				minVersion, err := semver.NewVersion(minVersionStr)
				if err != nil {
					return microerror.Mask(err)
				}

				if currentVersion.LessThan(minVersion) {
					release.Spec.Components[i].Version = minVersion.String()
				}
				break
			}
		}

		if found {
			continue
		}

		for i, app := range release.Spec.Apps {
			if app.Name == appname {
				found = true
				currentVersion, err := semver.NewVersion(app.Version)
				if err != nil {
					return microerror.Mask(err)
				}

				minVersion, err := semver.NewVersion(minVersionStr)
				if err != nil {
					return microerror.Mask(err)
				}

				if currentVersion.LessThan(minVersion) {
					release.Spec.Apps[i].Version = minVersion.String()
				}
				break
			}
		}

		if !found {
			r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("found unknown component %q from requests.yaml while it's not present in latest release.yaml; skipping", appname))
		}
	}

	return nil
}

// The mergeRequirements func takes a requests file and computes the list of applications and their minimum version.
func (r *runner) mergeRequirements(ctx context.Context, reqs requests, targetVersion semver.Version) (map[string]string, error) {
	apps := map[string]string{}

	for _, req := range reqs.Releases {
		// Check if the release is affected.
		constraint, err := semver.NewConstraint(req.Name)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if constraint.Check(&targetVersion) {
			for _, appreq := range req.Requests {
				if !strings.HasPrefix(appreq.Version, ">=") {
					r.logger.LogCtx(ctx, "level", "warning", "msg", fmt.Sprintf("Unable to parse app request %q: doesn't start with '>='", appreq.Version))
					continue
				}

				candidate, err := semver.NewVersion(strings.TrimSpace(strings.TrimPrefix(appreq.Version, ">=")))
				if err != nil {
					r.logger.LogCtx(ctx, "level", "warning", "msg", fmt.Sprintf("Unable to parse app request %q: invalid version. %s", appreq.Version, err))
					continue
				}

				current := apps[appreq.Name]
				if current == "" {
					// First request for this app/component.
					apps[appreq.Name] = candidate.String()
				} else {
					currentVersion, err := semver.NewVersion(strings.TrimSpace(strings.TrimPrefix(current, ">=")))
					if err != nil {
						return nil, microerror.Mask(err)
					}

					if candidate.GreaterThan(currentVersion) {
						apps[appreq.Name] = candidate.String()
					}
				}
			}
		}
	}

	return apps, nil
}
