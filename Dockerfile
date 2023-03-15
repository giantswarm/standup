FROM quay.io/giantswarm/alpine:3.16.2 AS binaries

ARG KUBECTL_VERSION=1.18.9
ARG GSCTL_VERSION=0.24.4

RUN apk add --no-cache ca-certificates curl \
    && mkdir -p /binaries \
    && curl -SL https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl -o /binaries/kubectl \
    && curl -SL https://github.com/giantswarm/gsctl/releases/download/${GSCTL_VERSION}/gsctl-${GSCTL_VERSION}-linux-amd64.tar.gz | \
       tar -C /binaries --strip-components 1 -xvzf - gsctl-${GSCTL_VERSION}-linux-amd64/gsctl \
    && chmod +x /binaries/*

FROM quay.io/giantswarm/alpine:3.16.2 as base

USER root
RUN apk add --no-cache git jq

RUN adduser -D giantswarm
USER giantswarm
COPY --from=binaries /binaries/* /usr/bin/
COPY ./standup /usr/local/bin/standup

ENTRYPOINT ["standup"]
