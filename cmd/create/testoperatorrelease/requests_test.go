package testoperatorrelease

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/micrologger"
)

func TestMergeRequirements(t *testing.T) {
	testCases := []struct {
		name          string
		targetVersion string
		requests      requests
		expected      map[string]string
		errorMatcher  func(error) bool
	}{
		{
			name:          "case 0: >= app specified twice",
			targetVersion: "13.0.0",
			requests: requests{
				Releases: []request{
					{
						Name: ">= 13.0.0",
						Requests: []apprequest{
							{
								Name:    "app-operator",
								Version: ">= 1.1.0",
							},
						},
					},
					{
						Name: ">= 13.0.0",
						Requests: []apprequest{
							{
								Name:    "app-operator",
								Version: ">= 1.0.0",
							},
						},
					},
				},
			},
			expected: map[string]string{
				"app-operator": "1.1.0",
			},
			errorMatcher: nil,
		},
		{
			name:          "case 1: >= app specified once",
			targetVersion: "13.0.0",
			requests: requests{
				Releases: []request{
					{
						Name: ">= 13.0.0",
						Requests: []apprequest{
							{
								Name:    "app-operator",
								Version: ">= 1.1.0",
							},
						},
					},
				},
			},
			expected: map[string]string{
				"app-operator": "1.1.0",
			},
			errorMatcher: nil,
		},
		{
			name:          "case 2: >= two different apps",
			targetVersion: "13.0.0",
			requests: requests{
				Releases: []request{
					{
						Name: ">= 13.0.0",
						Requests: []apprequest{
							{
								Name:    "app-operator",
								Version: ">= 1.1.0",
							},
						},
					},
					{
						Name: ">= 13.0.0",
						Requests: []apprequest{
							{
								Name:    "azure-operator",
								Version: ">= 2.1.0",
							},
						},
					},
				},
			},
			expected: map[string]string{
				"app-operator":   "1.1.0",
				"azure-operator": "2.1.0",
			},
			errorMatcher: nil,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			version, err := semver.NewVersion(tc.targetVersion)
			if err != nil {
				panic(err)
			}

			logger, err := micrologger.New(micrologger.Config{
				IOWriter: nil,
			})
			if err != nil {
				panic(err)
			}

			r := runner{
				logger: logger,
			}

			result, err := r.mergeRequirements(context.Background(), tc.requests, *version)

			switch {
			case err == nil && tc.errorMatcher == nil:
				// correct; carry on
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if !reflect.DeepEqual(tc.expected, result) {
				t.Fatalf("\n\nExpected %v, got %v\n", tc.expected, result)
			}
		})
	}
}
