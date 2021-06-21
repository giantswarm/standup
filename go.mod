module github.com/giantswarm/standup

go 1.14

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535 // indirect
	github.com/giantswarm/apiextensions/v2 v2.6.2
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/errors v0.3.0
	github.com/giantswarm/k8sclient/v4 v4.1.0
	github.com/giantswarm/microerror v0.3.0
	github.com/giantswarm/micrologger v0.5.0
	github.com/go-openapi/errors v0.19.6 // indirect
	github.com/go-openapi/swag v0.19.9 // indirect
	github.com/google/go-cmp v0.5.6
	github.com/mailru/easyjson v0.7.1 // indirect
	github.com/mitchellh/mapstructure v1.3.2 // indirect
	github.com/spf13/cobra v1.1.3
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	sigs.k8s.io/controller-runtime v0.9.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/coreos/etcd v3.3.10+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/coreos/etcd v3.3.13+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/gogo/protobuf v1.3.1 => github.com/gogo/protobuf v1.3.2
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.10-gs
)
