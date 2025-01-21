FROM golang:1.7.0
MAINTAINER Zhang Yifa <yzhang3@linkernetworks.com>

USER root
WORKDIR /usr/local/bin
EXPOSE 10022


RUN apt-get install git
RUN git clone https://github.com/LinkerNetworks/gotty.git $GOPATH/src/github.com/LinkerNetworks

RUN go get -d github.com/fsouza/go-dockerclient \
 && go get -d github.com/spf13/cobra \
 && go get -d github.com/zyfdegh/go-dockerpty \
 && go get -d github.com/jteeuwen/go-bindata/... \
 && go get -d github.com/tools/godep \
 && go install github.com/tools/godep \
 && go get -d github.com/zyfdegh/boomer \
 && go get -d github.com/LinkerNetworks/gotty/app \
 && go get -u github.com/jteeuwen/go-bindata/...

ENV PATH=${PATH}:${GOPATH}/bin
RUN ls $GOPATH/bin && whereis go-bindata

RUN cd $GOPATH/src/github.com/LinkerNetworks/gotty \
 && git submodule sync \
 && git submodule update --init --recursive \
 && godep restore \
 && make \
 && mv gotty /usr/local/bin/gotty

RUN git clone https://github.com/LinkerNetworks/remote-docker-exec.git $GOPATH/src/github.com/remote-docker-exec \
 && cd $GOPATH/src/github.com/remote-docker-exec \
 && go build -o /usr/local/bin/remote-docker-exec


# fix library dependencies, otherwise 'not found' error may occur when executing golang binary.
#RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

# volume will be mount to host
RUN mkdir -p /usr/local/bin/certs

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
COPY gotty.conf /usr/local/bin/gotty.conf
#COPY /usr/local/bin/docker-exec/remote-docker-exec /usr/local/bin/remote-docker-exec
#COPY /usr/local/bin/gotty/gotty /usr/local/bin/gotty

RUN chmod +x /usr/local/bin/remote-docker-exec \
                        /usr/local/bin/gotty \
                        /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

#### STAGE 2 #####
# Docker >= 17.05.0-ce
FROM linkerrepository/alpine:3.6
WORKDIR /usr/local/bin
EXPOSE 10022

COPY --from=0 /usr/local/bin/ /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
