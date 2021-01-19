package testoperatorrelease

import (
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func Test_generateReleaseName(t *testing.T) {
	t.Skip("timing sensitive tests; only enable on-demand locally")

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "case 0: simple release version",
			input:    "v13.0.0",
			expected: "v13.0.0-" + strconv.Itoa(int(time.Now().Unix())),
		},
		{
			name:     "case 1: alpha release version",
			input:    "v13.0.0-alpha3",
			expected: "v13.0.0-" + strconv.Itoa(int(time.Now().Unix())),
		},
		{
			name:     "case 2: beta release version",
			input:    "v13.0.0-beta1",
			expected: "v13.0.0-" + strconv.Itoa(int(time.Now().Unix())),
		},
		{
			name:     "case 3: non-standard release version",
			input:    "v13.0",
			expected: "v13.0-" + strconv.Itoa(int(time.Now().Unix())),
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			output := generateReleaseName(tc.input)

			if !cmp.Equal(output, tc.expected) {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.expected, output))
			}
		})
	}
}
