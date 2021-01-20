package testoperatorrelease

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
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

		releaseYAML, err := ioutil.ReadFile(requestsPath)
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
			r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("Adding a new component from the requirements is not currently supported. Tried to add %s", appname))
		}
	}

	m, _ := yaml.Marshal(release)
	fmt.Println(string(m))

	return nil
}

func affects(r request, targetVersion semver.Version) (bool, error) {
	lemmas := strings.Split(r.Name, ",")

	for _, lemma := range lemmas {
		lemma := strings.TrimSpace(lemma)

		firstChar := lemma[0:1]
		if firstChar == ">" || firstChar == "<" || firstChar == "=" {
			// > 13.0.2, < 12.0.0, >= 13.0.0, <= 13.0.1, = 13.0.0
			tokens := strings.Split(lemma, " ")
			if len(tokens) != 2 {
				return false, microerror.Maskf(invalidRequestError, "Unable to parse lemma %q", lemma)
			}
			operator := tokens[0]
			// We assume the version in this case is without wildcards.
			version, err := semver.NewVersion(tokens[1])
			if err != nil {
				return false, microerror.Mask(err)
			}

			switch operator {
			case ">":
				// Does not match if it is <=.
				if targetVersion.LessThan(version) || targetVersion.Equal(version) {
					return false, nil
				}
			case ">=":
				// Does not match if it is <.
				if targetVersion.LessThan(version) {
					return false, nil
				}
			case "<":
				// Does not match if it is >=.
				if targetVersion.GreaterThan(version) || targetVersion.Equal(version) {
					return false, nil
				}
			case "<=":
				// Does not match if it is >.
				if targetVersion.GreaterThan(version) {
					return false, nil
				}
			case "=":
				// Does not match if it is !=.
				if !targetVersion.Equal(version) {
					return false, nil
				}
			default:
				return false, microerror.Maskf(invalidRequestError, "Unrecognized operator %q", operator)
			}
		} else {
			// Wildcard things like "12.*"
			reg := strings.ReplaceAll(lemma, ".", "\\.")
			reg = strings.ReplaceAll(reg, "*", ".*")
			re := regexp.MustCompile(reg)

			if !re.Match([]byte(targetVersion.String())) {
				return false, nil
			}
		}
	}

	return true, nil
}

func (r *runner) mergeRequirements(ctx context.Context, reqs requests, targetVersion semver.Version) (map[string]string, error) {
	apps := map[string]string{}

	for _, req := range reqs.Releases {
		// Check if the release is affected.
		affected, err := affects(req, targetVersion)
		if err != nil {
			r.logger.LogCtx(ctx, "level", "warning", "msg", fmt.Sprintf("Unable to check if rule named %q affects version %q: %s", req.Name, targetVersion.String(), err))
			continue
		}

		if affected {
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
