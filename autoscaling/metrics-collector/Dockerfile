FROM golang:1.8
ENV PROJECT $GOPATH/src/linkernetworks.com/dcos-backend/autoscaling/metrics-collector
WORKDIR $PROJECT

# install govendor
RUN go get github.com/kardianos/govendor

COPY . $PROJECT

RUN make get-dep && \
    go test $(go list ./... | grep -v /vendor/) && \
    go build -o /bin/metrics-collector && \
    rm -rf $PROJECT

#### STAGE 2 ####
# Docker >= 17.05.0-ce
FROM linkerrepository/alpine:3.6

COPY --from=0 /bin/metrics-collector /bin/metrics-collector

EXPOSE 10005
CMD ["/bin/metrics-collector"]
