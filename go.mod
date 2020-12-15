module github.com/giantswarm/standup

go 1.14

require (
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/giantswarm/apiextensions/v3 v3.4.0
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/errors v0.2.3
	github.com/giantswarm/k8sclient/v5 v5.0.0
	github.com/giantswarm/k8sportforward/v2 v2.0.0
	github.com/giantswarm/microerror v0.2.1
	github.com/giantswarm/micrologger v0.3.3
	github.com/go-openapi/runtime v0.19.20 // indirect
	github.com/google/go-cmp v0.5.2
	github.com/prometheus/alertmanager v0.21.0
	github.com/prometheus/client_golang v1.6.0
	github.com/spf13/cobra v1.0.0
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	k8s.io/api v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v0.18.9
	k8s.io/utils v0.0.0-20200731180307-f00132d28269 // indirect
	sigs.k8s.io/yaml v1.2.0
)

replace sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.10-gs
