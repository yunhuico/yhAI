default: help

help:
	@echo -e "Select a sub command \n"
	@echo -e "build: \n\t Build dcos-client Docker image"
	@echo -e "run: \n\t Run dcos-client in Docker container"
	@echo -e "push: \n\t Push image to DockerHub"
	@echo -e "fmt: \n\t Format source code with go fmt"
	@echo -e "help: \n\t Display this help"
	@echo -e "\n"
	@echo -e "See README.md for more."

IMAGE_NAME="linkerrepository/linkerdcos_client:dev"

build:
	cp -r ../common ./common.tmp
	docker build -t ${IMAGE_NAME} .
	rm -rf common.tmp

run:
	docker run --rm --network=host \
		${IMAGE_NAME}

push:
	docker push ${IMAGE_NAME}

fmt:
	go fmt $(go list ./... | grep -v /vendor/)
