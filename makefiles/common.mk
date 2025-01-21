GO111MODULE ?= on
GOPROXY ?= https://proxy.golang.org
GOSUMDB ?= sum.golang.org

TIME_LONG	= `date +%Y-%m-%d' '%H:%M:%S`
TIME_SHORT	= `date +%H:%M:%S`
TIME		= $(TIME_SHORT)

BLUE         := $(shell printf "\033[34m")
YELLOW       := $(shell printf "\033[33m")
RED          := $(shell printf "\033[31m")
GREEN        := $(shell printf "\033[32m")
CNone        := $(shell printf "\033[0m")

INFO	= echo ${TIME} ${BLUE}[ .. ]${CNone}
WARN	= echo ${TIME} ${YELLOW}[WARN]${CNone}
ERR		= echo ${TIME} ${RED}[FAIL]${CNone}
OK		= echo ${TIME} ${GREEN}[ OK ]${CNone}
FAIL	= (echo ${TIME} ${RED}[FAIL]${CNone} && false)

CGO ?= 0
GOLANGCILINT_VERSION ?= latest
# UltraFox version
ULTRAFOX_VERSION ?= $(shell git describe --tags --dirty)
GIT_COMMIT          ?= git-$(shell git rev-parse --short HEAD)
GIT_COMMIT_LONG     ?= $(shell git rev-parse HEAD)
ULTRAFOX_VERSION_KEY    := jihulab.com/jihulab/ultrafox/ultrafox/version.UltrafoxVersion
ULTRAFOX_GITVERSION_KEY := jihulab.com/jihulab/ultrafox/ultrafox/version.GitRevision
LDFLAGS             ?= "-s -w -X $(ULTRAFOX_VERSION_KEY)=$(ULTRAFOX_VERSION) -X $(ULTRAFOX_GITVERSION_KEY)=$(GIT_COMMIT)"


BIN_PATH=./bin/ultrafox

# for multiarch build
LOCAL_ARCH := $(shell uname -m)
ifeq ($(LOCAL_ARCH),x86_64)
	TARGET_ARCH_LOCAL=amd64
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 5),armv8)
	TARGET_ARCH_LOCAL=arm64
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 7),aarch64)
	TARGET_ARCH_LOCAL=arm64
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 5),arm64)
	TARGET_ARCH_LOCAL=arm64
else
	TARGET_ARCH_LOCAL=amd64
endif
export GOARCH ?= $(TARGET_ARCH_LOCAL)

LOCAL_OS := $(shell uname)
ifeq ($(LOCAL_OS),Linux)
   TARGET_OS_LOCAL = linux
else ifeq ($(LOCAL_OS),Darwin)
   TARGET_OS_LOCAL = darwin
else
   TARGET_OS_LOCAL ?= windows
endif
export GOOS ?= $(TARGET_OS_LOCAL)

# Default docker container and e2e test target.
TARGET_OS ?= linux
TARGET_ARCH ?= amd64
