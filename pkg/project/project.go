package project

var (
	description = "Command line tool for testing clusters."
	gitSHA      = "n/a"
	name        = "standup"
	source      = "https://github.com/giantswarm/standup"
	version     = "2.0.0"
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
