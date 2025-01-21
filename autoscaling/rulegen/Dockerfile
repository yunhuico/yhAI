## Multi-stage needs Linux Docker >= 17.05.0-ce

#### STAGE 0 ####
FROM golang:1.8
ENV PROJECT $GOPATH/src/linkernetworks.com/dcos-backend/autoscaling/rulegen/
WORKDIR $PROJECT

RUN go get github.com/kardianos/govendor

COPY . $PROJECT
RUN make get-dep && \
    go test $(go list ./... | grep -v /vendor/) && \
    go build -o /go/bin/rulegen && \
    rm -rf $PROJECT

#### STAGE 1 ####
## Default port 8080, set container env LISTEN_PORT to change it
FROM linkerrepository/alpine:3.6

COPY --from=0 /go/bin/rulegen /rulegen

CMD /rulegen
