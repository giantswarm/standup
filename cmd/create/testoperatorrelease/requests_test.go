package testoperatorrelease

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/micrologger"
)

func TestAffects(t *testing.T) {
	testCases := []struct {
		name          string
		targetVersion string
		request       request
		expected      bool
		errorMatcher  func(error) bool
	}{
		{
			name:          "case 0: >= with the target version being =",
			targetVersion: "13.0.0",
			request:       request{Name: ">= 13.0.0"},
			expected:      true,
			errorMatcher:  nil,
		},
		{
			name:          "case 1: >= with the target version being >",
			targetVersion: "13.0.1",
			request:       request{Name: ">= 13.0.0"},
			expected:      true,
			errorMatcher:  nil,
		},
		{
			name:          "case 2: >= with the target version being <",
			targetVersion: "13.0.0",
			request:       request{Name: ">= 13.0.1"},
			expected:      false,
			errorMatcher:  nil,
		},
		{
			name:          "case 3: <= with the target version being =",
			targetVersion: "13.0.0",
			request:       request{Name: "<= 13.0.0"},
			expected:      true,
			errorMatcher:  nil,
		},
		{
			name:          "case 4: <= with the target version being <",
			targetVersion: "12.1.2",
			request:       request{Name: "<= 13.0.0"},
			expected:      true,
			errorMatcher:  nil,
		},
		{
			name:          "case 5: <= with the target version being >",
			targetVersion: "13.0.1",
			request:       request{Name: "<= 13.0.0"},
			expected:      false,
			errorMatcher:  nil,
		},
		{
			name:          "case 6: = with the target version being =",
			targetVersion: "13.0.0",
			request:       request{Name: "= 13.0.0"},
			expected:      true,
			errorMatcher:  nil,
		},
		{
			name:          "case 7: = with the target version being !=",
			targetVersion: "13.0.1",
			request:       request{Name: "= 13.0.0"},
			expected:      false,
			errorMatcher:  nil,
		},
		{
			name:          "case 8: no operator, fixed version",
			targetVersion: "13.0.0",
			request:       request{Name: "13.0.0"},
			expected:      true,
			errorMatcher:  nil,
		},
		{
			name:          "case 9: no operator, fixed version not matching",
			targetVersion: "13.0.1",
			request:       request{Name: "13.0.0"},
			expected:      false,
			errorMatcher:  nil,
		},
		{
			name:          "case 10: wildcard matching patch",
			targetVersion: "13.0.1",
			request:       request{Name: "13.0.*"},
			expected:      true,
			errorMatcher:  nil,
		},
		{
			name:          "case 11: wildcard matching minor",
			targetVersion: "13.0.1",
			request:       request{Name: "13.*"},
			expected:      true,
			errorMatcher:  nil,
		},
		{
			name:          "case 12: wildcard matching major",
			targetVersion: "13.0.1",
			request:       request{Name: "*"},
			expected:      true,
			errorMatcher:  nil,
		},
		{
			name:          "case 13: wildcard not matching minor",
			targetVersion: "13.1.1",
			request:       request{Name: "13.0.*"},
			expected:      false,
			errorMatcher:  nil,
		},
		{
			name:          "case 14: wildcard not matching major",
			targetVersion: "12.0.1",
			request:       request{Name: "13.*"},
			expected:      false,
			errorMatcher:  nil,
		},
		{
			name:          "case 15: wildcard only",
			targetVersion: "12.0.1",
			request:       request{Name: "*"},
			expected:      true,
			errorMatcher:  nil,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			version, err := semver.NewVersion(tc.targetVersion)
			if err != nil {
				panic(err)
			}

			result, err := affects(tc.request, *version)

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

			if tc.expected != result {
				t.Fatalf("\n\nExpected %t, got %t\n", tc.expected, result)
			}
		})
	}
}

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
				"app-operator": "1.0.0",
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
				"app-operator":   "1.0.0",
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

			if reflect.DeepEqual(tc.expected, result) {
				t.Fatalf("\n\nExpected %v, got %v\n", tc.expected, result)
			}
		})
	}
}
