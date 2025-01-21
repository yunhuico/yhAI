FROM golang:1.8.3
MAINTAINER linkernetworks.com

USER root

RUN apt-get install git

ENV REPO $GOPATH/src/github.com/google/cadvisor

ENV branch=linker

RUN git clone -b $branch https://github.com/LinkerNetworks/cadvisor --depth=1 $REPO

RUN go get github.com/tools/godep
RUN cd $GOPATH/src/github.com/tools/godep && \
	go build .

RUN cd $REPO \
	&& godep restore \
    && godep go build .

RUN cp $REPO/cadvisor /usr/local/bin/cadvisor && \
    chmod +x /usr/local/bin/cadvisor

EXPOSE 10000
ENTRYPOINT ["/usr/local/bin/cadvisor", "--port", "10000"]

#### STAGE 2 ####
# Docker >= 17.05.0-ce
FROM linkerrepository/alpine:3.6

COPY --from=0 /usr/local/bin/cadvisor /usr/local/bin/cadvisor

EXPOSE 10000
ENTRYPOINT ["/usr/local/bin/cadvisor", "--port", "10000"]
