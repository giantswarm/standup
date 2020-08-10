# Need to build a branch of gsctl until changes are merged
FROM quay.io/giantswarm/golang:1.14.7 AS gsctl
RUN git clone https://github.com/giantswarm/gsctl.git
WORKDIR /go/gsctl
RUN git checkout add-json-output
RUN CGO_ENABLED=0 go build

# Use the giantswarm alpine again when gsctl changes are merged
FROM quay.io/giantswarm/alpine:3.11-giantswarm as base

COPY --from=gsctl /go/gsctl/gsctl /usr/bin/gsctl

ADD ./standup /standup

ENTRYPOINT ["/standup"]
