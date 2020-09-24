package git

import (
	"os/exec"
	"strings"

	"github.com/giantswarm/microerror"
)

func Diff(dir, ref string) (string, error) {
	// Determine the files added in this branch not in master
	argsArr := []string{
		"diff",
		"--name-status",   // only show filename and the type of change (A=added, etc.)
		ref,               // diff against the passed reference
		"--diff-filter=A", // only show added files
		"--no-renames",    // disable rename detection so we always find new releases
		"HEAD",            // base ref for the diff
	}
	diff, err := runGit(argsArr, dir)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return diff, nil
}

func Fetch(dir string) error {
	// Fetch master so we can diff against it
	argsArr := []string{
		"fetch",
		"--unshallow",
		"origin",
		"master",
	}
	_, err := runGit(argsArr, dir)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func MergeBase(dir string) (string, error) {
	// Fetch master so we can diff against it
	argsArr := []string{
		"merge-base",
		"HEAD",
		"origin/master",
	}
	mergeBase, err := runGit(argsArr, dir)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return strings.TrimSpace(mergeBase), nil
}

func runGit(args []string, dir string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", microerror.Mask(err)
	}
	return string(output), nil
}
