[TOC]

# Summary
This document explains how to build dcos_client binary in local.

To build dcos_client in Docker image.

```sh
cd ..
docker build -t client -f Dockerfile.client .
```

# Prerequisites

* Go 1.6+

# Clone
Clone this project under GOPATH.

```sh
git clone git@bitbucket.org:linkernetworks/linker-dcos.git \
	$GOPATH/src/linkernetworks.com/
```

Rename project.

```sh
cd $GOPATH/src/linkernetworks.com/dcos-backend/client
mv linker-dcos dcos-backend
```

# Build
```sh
cd dcos-backend/client
./build.sh
```

# Run
The binary file `dcos_client` are built and copied to `$GOPATH/bin/`,
along with the config file `dcos_client.properties`.

```sh
cd $GOPATH/bin
# edit config file
vim dcos_client.properties

./dcos_client
```
