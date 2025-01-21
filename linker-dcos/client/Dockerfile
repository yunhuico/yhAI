# This Dockerfile is used for development ONLY.
# To see the production image is built, redirect to ../Dockerfile.client

FROM golang:1.8

ENV PROJECT $GOPATH/src/linkernetworks.com/dcos-backend/client
WORKDIR $PROJECT

# get deps
RUN go get -d github.com/Sirupsen/logrus && \
    go get -d github.com/magiconair/properties && \
    go get -d github.com/emicklei/go-restful && \
    go get -d github.com/emicklei/go-restful-swagger12 && \
    go get -d github.com/compose/mejson && \
    go get -d github.com/LinkerNetworks/go-marathon && \
    go get -d github.com/jmoiron/jsonq && \
    go get -d github.com/pborman/uuid && \
    go get -d github.com/bmizerany/assert

# install swagger
RUN git clone https://github.com/wordnik/swagger-ui.git /linker/swagger-ui && \
    sed -i 's/"http:\/\/petstore.swagger.io\/v2\/swagger.json"/"\/apidocs.json"/g' /linker/swagger-ui/dist/index.html

# build binary
COPY . $PROJECT
RUN mv common.tmp ../common && \
    go test ./... && \
    go build -o /linker/dcos_client

# copy config file and scripts
RUN cp -r ./resources/* /linker/ && \
    cp ./dcos_client.properties /linker/ && \
    cp ./entrypoint.sh /linker/

RUN chmod u+x /linker/*.sh && \
    chmod u+x /linker/smack/*.sh

# delete source code
RUN rm -rf $PROJECT

EXPOSE 10004

WORKDIR /linker

CMD ["/linker/dcos_client"]
