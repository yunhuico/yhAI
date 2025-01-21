#FROM jiezhang/swarm:1.2.4
FROM golang:1.5.4-alpine

ARG GOOS

COPY ./certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY ./tmp /tmp

ENV GO15VENDOREXPERIMENT=1

RUN apk add --no-cache --virtual git

RUN git clone -b v1.2.4-rc3 https://github.com/LinkerNetworks/swarm /go/src/github.com/docker/swarm

WORKDIR /go/src/github.com/docker/swarm

RUN ls /

RUN set -ex \
	&& apk add --no-cache --virtual .build-deps \
	&& GOOS=$GOOS CGO_ENABLED=0 go install -v -a -tags netgo -installsuffix netgo -ldflags "-w -X github.com/docker/swarm/version.GITCOMMIT `git rev-parse --short HEAD` -X github.com/docker/swarm/version.BUILDTIME \"`date -u`\""  \
	&& GOOS=$GOOS CGO_ENABLED=0 go build -o swarm \
	&& apk del .build-deps


#RUN cp swarm /swarm

#### STAGE 2 ####
FROM linkerrepository/alpine:3.6

COPY ./tmp /tmp
COPY --from=0 /go/src/github.com/docker/swarm/swarm /swarm

ENV SWARM_HOST :2375
EXPOSE 2375

VOLUME /.swarm

ENTRYPOINT ["/swarm"]
CMD ["--help"]
