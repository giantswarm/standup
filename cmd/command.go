package cmd

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/standup/cmd/cleanup"
	"github.com/giantswarm/standup/cmd/create"
	"github.com/giantswarm/standup/cmd/version"
	"github.com/giantswarm/standup/cmd/wait"
)

const (
	name        = "standup"
	description = "Provides commands for managing test clusters and running tests against them."
)

type Config struct {
	Logger micrologger.Logger
	Stderr io.Writer
	Stdout io.Writer

	BinaryName string
	GitCommit  string
	Source     string
}

func New(config Config) (*cobra.Command, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}

	if config.GitCommit == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.GitCommit must not be empty", config)
	}
	if config.Source == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Source must not be empty", config)
	}

	var err error

	var cleanupCmd *cobra.Command
	{
		c := cleanup.Config{
			Logger: config.Logger,
			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		cleanupCmd, err = cleanup.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var createCmd *cobra.Command
	{
		c := create.Config{
			Logger: config.Logger,
			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		createCmd, err = create.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var versionCmd *cobra.Command
	{
		c := version.Config{
			Logger: config.Logger,
			Stderr: config.Stderr,
			Stdout: config.Stdout,

			GitCommit: config.GitCommit,
			Source:    config.Source,
		}

		versionCmd, err = version.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var waitCmd *cobra.Command
	{
		c := wait.Config{
			Logger: config.Logger,
			Stderr: config.Stderr,
			Stdout: config.Stdout,
		}

		waitCmd, err = wait.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	f := &flag{}

	r := &runner{
		flag:   f,
		logger: config.Logger,
		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:          name,
		Short:        description,
		Long:         description,
		RunE:         r.Run,
		SilenceUsage: true,
	}

	f.Init(c)

	c.AddCommand(cleanupCmd)
	c.AddCommand(createCmd)
	c.AddCommand(versionCmd)
	c.AddCommand(waitCmd)

	return c, nil
}
