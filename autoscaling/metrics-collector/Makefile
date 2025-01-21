default: help

help:
	@echo -e "Select a sub command \n"
	@echo -e "install-vendor: \n\t Install govendor"
	@echo -e "init-vendor: \n\t Init vendor/vendor.json"
	@echo -e "update-dep: \n\t Remove unused packages and add new packages in vendor/"
	@echo -e "get-dep: \n\t Synchronize packages from GOPATH or download online"
	@echo -e "build: \n\t Build metrics-collector Docker image"
	@echo -e "run: \n\t Run metrics-collector Docker container"
	@echo -e "push: \n\t Push metrics-collector image to DockerHub"
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

build:
	docker build -t linkerrepository/metrics-collector .

run:
	docker run --rm -p 10005:10005 \
		-e DAEMON_MODE=${DAEMON_MODE} \
		-e CADVISORS="${CADVISORS}" \
		-e CADVISOR_TIMEOUT=${CADVISOR_TIMEOUT} \
		-e POLLING_SEC=${POLLING_SEC} \
		-e ENABLE_UPDATER=${ENABLE_UPDATER} \
		-e ADDR_UPDATE_SEC=${ADDR_UPDATE_SEC} \
		-e MESOS_ENDPOINT=${MESOS_ENDPOINT} \
		-e CADVISOR_PORT=${CADVISOR_PORT} \
		linkerrepository/metrics-collector

push:
	docker push linkerrepository/metrics-collector

fmt:
	go fmt $(go list ./... | grep -v /vendor/)
