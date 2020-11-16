package project

var (
	description = "Command line tool for testing clusters."
	gitSHA      = "n/a"
	name        = "standup"
	source      = "https://github.com/giantswarm/standup"
	version     = "1.1.2-dev"
)

func Description() string {
	return description
}

func GitSHA() string {
	return gitSHA
}

func Name() string {
	return name
}

func Source() string {
	return source
}

func Version() string {
	return version
}
