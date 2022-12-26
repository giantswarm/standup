module github.com/giantswarm/standup

go 1.14

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535 // indirect
	github.com/giantswarm/apiextensions/v2 v2.6.2
	github.com/giantswarm/backoff v1.0.0
	github.com/giantswarm/errors v0.3.0
	github.com/giantswarm/k8sclient/v4 v4.1.0
	github.com/giantswarm/microerror v0.4.0
	github.com/giantswarm/micrologger v0.6.0
	github.com/go-openapi/errors v0.19.6 // indirect
	github.com/google/go-cmp v0.5.9
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/spf13/cobra v1.6.0
	k8s.io/api v0.26.0
	k8s.io/apimachinery v0.26.0
	k8s.io/client-go v0.26.0
	sigs.k8s.io/controller-runtime v0.14.1
	sigs.k8s.io/yaml v1.3.0
)

replace (
	github.com/aws/aws-sdk-go => github.com/aws/aws-sdk-go v1.44.66
	github.com/coreos/etcd v3.3.10+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/coreos/etcd v3.3.13+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt v3.2.1+incompatible
	github.com/gobuffalo/packr/v2 => github.com/gobuffalo/packr/v2 v2.8.3
	github.com/gogo/protobuf v1.3.1 => github.com/gogo/protobuf v1.3.2
	github.com/gorilla/websocket v1.4.0 => github.com/gorilla/websocket v1.4.2
	github.com/miekg/dns => github.com/miekg/dns v1.1.50
	github.com/pkg/sftp => github.com/pkg/sftp v1.13.5
	github.com/prometheus/client_golang v1.11.0 => github.com/prometheus/client_golang v1.12.2
	github.com/prometheus/client_golang v1.5.1 => github.com/prometheus/client_golang v1.12.2
	go.mongodb.org/mongo-driver v1.3.4 => go.mongodb.org/mongo-driver v1.10.0
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.10-gs
)
