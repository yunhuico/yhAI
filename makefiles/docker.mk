DOCKER:=docker
DOCKERFILE=Dockerfile
DOCKERFILE_DIR?=./docker
#CI_COMMIT_TAG?=latest

ULTRAFOX_IMAGE_REGISTRY=registry.jihulab.com


ifeq ($(TARGET_ARCH),arm)
DOCKER_IMAGE_PLATFORM:=$(TARGET_OS)/arm
else ifeq ($(TARGET_ARCH),arm64)
DOCKER_IMAGE_PLATFORM:=$(TARGET_OS)/arm64/v8
else
DOCKER_IMAGE_PLATFORM:=$(TARGET_OS)/amd64
endif

# Supported docker image architecture
DOCKERMUTI_ARCH=linux-amd64 #linux-arm64 #linux-arm

ifeq ($(CI_COMMIT_TAG),)
RELEASE_TAG:=$(CI_COMMIT_SHORT_SHA)
ULTRAFOX_IMAGE_NAME:=jihulab/ultrafox/ultrafox/$(CI_COMMIT_REF_SLUG)
else
RELEASE_TAG:=$(CI_COMMIT_TAG)
ULTRAFOX_IMAGE_NAME:=jihulab/ultrafox/ultrafox
endif

DOCKER_IMAGE_TAG=$(ULTRAFOX_IMAGE_REGISTRY)/$(ULTRAFOX_IMAGE_NAME):$(RELEASE_TAG)

export DOCKER_CLI_EXPERIMENTAL=enabled

check-arch:
ifeq ($(TARGET_OS),)
	$(error TARGET_OS environment variable must be set)
endif
ifeq ($(TARGET_ARCH),)
	$(error TARGET_ARCH environment variable must be set)
endif

docker-build: check-arch
	$(info Building $(DOCKER_IMAGE_TAG) docker image ...)
	-$(DOCKER) context create build
	-$(DOCKER) buildx create build --name build --driver docker-container --use
	# -$(DOCKER) run --rm --privileged multiarch/qemu-user-static --reset -p yes
	$(DOCKER) buildx build --build-arg LDFLAGS=$(LDFLAGS) \
		--build-arg ASSETS_PATH=${ASSETS_PATH} \
		--cache-from type=registry,ref=$(DOCKER_IMAGE_TAG)-$(TARGET_OS)-$(TARGET_ARCH) \
		--platform $(DOCKER_IMAGE_PLATFORM) \
		-f $(DOCKERFILE_DIR)/$(DOCKERFILE) \
		-t $(DOCKER_IMAGE_TAG)-$(TARGET_OS)-$(TARGET_ARCH) .

# push docker image to the registry
docker-push:
	$(info Pushing $(DOCKER_IMAGE_TAG) docker image ...)
	-$(DOCKER) context create build
	-$(DOCKER) buildx create build --name build --driver docker-container --use
	# -$(DOCKER) run --rm --privileged multiarch/qemu-user-static --reset -p yes
	$(DOCKER) buildx build --build-arg LDFLAGS=$(LDFLAGS) \
		--build-arg ASSETS_PATH=${ASSETS_PATH} \
		--cache-from type=registry,ref=$(DOCKER_IMAGE_TAG)-$(TARGET_OS)-$(TARGET_ARCH) \
		--platform $(DOCKER_IMAGE_PLATFORM) \
		-f $(DOCKERFILE_DIR)/$(DOCKERFILE) \
		-t $(DOCKER_IMAGE_TAG)-$(TARGET_OS)-$(TARGET_ARCH) . --push

build-manifest:
	$(DOCKER) manifest create $(DOCKER_IMAGE_TAG) $(DOCKERMUTI_ARCH:%=$(DOCKER_IMAGE_TAG)-%)
	$(DOCKER) manifest push $(DOCKER_IMAGE_TAG)

image-digest:
	@echo DIGEST=`$(DOCKER) images --no-trunc --quiet $(DOCKER_IMAGE_TAG)`

image-tag:
	@echo "NEXTVERSION=\"$(RELEASE_TAG)\""
