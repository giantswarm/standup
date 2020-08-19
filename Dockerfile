# Need to build a branch of gsctl until changes are merged
FROM quay.io/giantswarm/golang:1.14.7 AS gsctl
RUN git clone https://github.com/giantswarm/gsctl.git
WORKDIR /go/gsctl
RUN git checkout add-json-output
RUN CGO_ENABLED=0 go build

FROM quay.io/giantswarm/alpine:3.12 AS kubectl
ARG VERSION=v1.18.8
RUN apk add --no-cache ca-certificates \
    && apk add --update -t deps curl \
    && curl https://storage.googleapis.com/kubernetes-release/release/$VERSION/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl \
    && chmod +x /usr/local/bin/kubectl

# Use the giantswarm alpine again when gsctl changes are merged
FROM quay.io/giantswarm/alpine:3.11-giantswarm as base

USER root
RUN apk add --no-cache git

USER giantswarm
COPY --from=gsctl /go/gsctl/gsctl /usr/bin/gsctl
COPY --from=kubectl /usr/local/bin/kubectl /usr/bin/kubectl
ADD ./standup /standup

ENTRYPOINT ["/standup"]
