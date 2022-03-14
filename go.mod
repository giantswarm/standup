module github.com/giantswarm/standup

go 1.14

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/giantswarm/apiextensions/v2 v2.6.2
	github.com/giantswarm/backoff v1.0.0
	github.com/giantswarm/errors v0.3.0
	github.com/giantswarm/k8sclient/v4 v4.1.0
	github.com/giantswarm/microerror v0.4.0
	github.com/giantswarm/micrologger v0.6.0
	github.com/go-openapi/errors v0.19.6 // indirect
	github.com/google/go-cmp v0.5.7
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/spf13/cobra v1.4.0
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/sys v0.0.0-20211205182925-97ca703d548d // indirect
	k8s.io/api v0.23.0
	k8s.io/apimachinery v0.23.0
	k8s.io/client-go v0.23.0
	sigs.k8s.io/controller-runtime v0.11.1
	sigs.k8s.io/yaml v1.3.0
)

replace (
	github.com/coreos/etcd v3.3.10+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/coreos/etcd v3.3.13+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt v3.2.1+incompatible
	github.com/gogo/protobuf v1.3.1 => github.com/gogo/protobuf v1.3.2
	github.com/gorilla/websocket v1.4.0 => github.com/gorilla/websocket v1.4.2
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.10-gs
)
