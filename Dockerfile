FROM quay.io/giantswarm/alpine:3.11-giantswarm

ADD ./standup /standup

ENTRYPOINT ["/standup"]
