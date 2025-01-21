include makefiles/common.mk

.PHONY: test
# Run tests
test: unit-test
	@$(OK) unit-tests pass

reviewable: fmt vet lint # Make your PR ready to review
	go mod tidy

vet:
	go vet ./...

fmt:
	go fmt ./...

unit-test:
	go test -v -race ./...

lint: golangci
	$(GOLANGCILINT) version
	$(GOLANGCILINT) run ./...

.PHONY: coverage
coverage: coverage-test build-coverage

build-coverage:
	mkdir -p out/cobertura
	go get github.com/boumenot/gocover-cobertura
	go run github.com/boumenot/gocover-cobertura < coverage.out > out/cobertura/cobertura-coverage.xml

coverage-test:
	# ignore data race when coverage.
	go test -coverpkg=./pkg/workflow -coverprofile=coverage.out $(shell go list ./... | grep -v /vendor/)
	@echo 'Generating TXT coverage report'
	mkdir -p out/
	@echo 'Generating HTML coverage report'
	go tool cover -o "out/coverprofile.html" -html="coverage.out"
	@echo 'General coverage percentage:'
	@go tool cover -func="coverage.out"

.PHONY: build
# Build binary
build: build-ultrafox
	@$(OK) build succeed

build-ultrafox:
	@echo "build ultrafox $(GOOS)/$(GOARCH)"
	time go build -o $(BIN_PATH) -ldflags $(LDFLAGS) ./cmd/ultrafox/main.go

build-cleanup:
	rm -rf bin

doc-gen:
	rm -rf ./docs/cli/*
	go run ./hack/docgen/gen.go

# depends goda and Graphviz
# go install github.com/loov/goda@master
draw-structure:
	goda graph ./... | dot -Tsvg -o graph.svg

.PHONY: golangci
golangci: install-golangci
ifneq ($(shell which golangci-lint),)
GOLANGCILINT=$(shell which golangci-lint)
else ifeq ($(shell which $(GOBIN)/golangci-lint),)
GOLANGCILINT=$(GOBIN)/golangci-lint
else
GOLANGCILINT=$(GOBIN)/golangci-lint
endif

.PHONY: install-golangci
install-golangci:
ifneq ($(shell which golangci-lint),)
	@$(OK) golangci-lint is already installed
else ifeq ($(shell which $(GOBIN)/golangci-lint),)
	@{ \
	set -e ;\
	echo 'installing golangci-lint-$(GOLANGCILINT_VERSION)' ;\
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) $(GOLANGCILINT_VERSION) ;\
	echo 'Successfully installed' ;\
	}
else
	@$(OK) golangci-lint is already installed
endif

build-swagger:
	swag init -d pkg/apiserver,pkg/model -g server.go --instanceName v1 --parseDependency -o ./docs/api_docs

################################################################################
# Target: docker                                                               #
################################################################################
include makefiles/docker.mk

