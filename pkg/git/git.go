package git

import (
	"os/exec"

	"github.com/giantswarm/microerror"
)

func Diff(dir string) (string, error) {
	// Determine the files added in this branch not in master
	argsArr := []string{
		"diff",
		"--name-status",   // only show filename and the type of change (A=added, etc.)
		"origin/master",   // diff against the latest master
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
		"origin",
		"master",
	}
	_, err := runGit(argsArr, dir)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
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
