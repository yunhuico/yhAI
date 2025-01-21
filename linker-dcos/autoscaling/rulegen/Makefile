default: help

help:
	@echo -e "Select a sub command \n"
	@echo -e "install-vendor: \n\t Install govendor"
	@echo -e "init-vendor: \n\t Init vendor/vendor.json"
	@echo -e "update-dep: \n\t Remove unused packages and add new packages in vendor/"
	@echo -e "get-dep: \n\t Synchronize packages from GOPATH or download online"
	@echo -e "build: \n\t Build binary"
	@echo -e "run: \n\t Run binary"
	@echo -e "build-docker: \n\t Build rulegen Docker image"
	@echo -e "run-docker: \n\t Run rulegen in Docker container"
	@echo -e "fmt: \n\t Format source code with go fmt"
	@echo -e "help: \n\t Display this help"
	@echo -e "\n"
	@echo -e "See README.md for more."

install-vendor:
	go get github.com/kardianos/govendor

init-vendor:
	govendor init

update-dep:
	govendor remove +unused
	govendor add +external

get-dep:
	govendor sync

OUTPUT=bin

build:
	go build -o ${OUTPUT}/rulegen

run:
	./${OUTPUT}/rulegen

test:
	go test ./...

fmt:
	go fmt $(go list ./... | grep -v /vendor/)

build-docker:
	docker build -t linkerrepository/rulegen:dev .

run-docker:
	docker run linkerrepository/rulegen:dev
