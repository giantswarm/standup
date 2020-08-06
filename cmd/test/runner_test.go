package test

import (
	"testing"
)

var diff = `A       aws/v13.0.0/README.md
A       aws/v13.0.0/kustomization.yaml
A       aws/v13.0.0/release.diff
A       aws/v13.0.0/release.yaml`

func Test_findNewRelease(t *testing.T) {
	path, provider, release, err := findNewRelease(diff)
	if err != nil {
		t.Fatal(err)
	}
	if path != "aws/v13.0.0/release.yaml" {
		t.Errorf("expected aws/v13.0.0/release.yaml, found %s", path)
	}
	if provider != "aws" {
		t.Errorf("expected aws, found %s", provider)
	}
	if release != "v13.0.0" {
		t.Errorf("expected v13.0.0, found %s", release)
	}
}
