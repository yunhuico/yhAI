# README #

# How to build mesos related files?

## Checkout code from github
```
$ git clone https://github.com/xinxian0458/mesos.git
$ cd mesos
$ git checkout -b 1.0.1-rc2 origin/1.0.1-rc2
```

## Build on-build image
```
$ docker build -t mesos-build .
```

## Build mesos in on-build container environment
```
$ docker run -it -v ${PWD}:/home/ubuntu/mesos mesos-build bash
$ ./bootstrap
$ mkdir build
$ cd build
$ ../configure CXXFLAGS=-Wno-deprecated-declarations
$ make
```

# Copy files to this folder
* mesos/build/src/.lib/libmesos-1.0.1.so
* mesos/build/src/.lib/libmesos.la
* mesos/build/src/.lib/libpostlaunchdockerhook-1.0.1.so
* mesos/build/src/.lib/libpostlaunchdockerhook.la

# Build mesos-slave-hook image
```
docker build -t linkerrepository/mesos-slave-hook:2.0.0-1.0.1 .
```
## clean mesos folder
```
sudo rm -rf aclocal.m4 ar-lib autom4te.cache build .clang-format compile config.guess config.sub configure depcomp .gitignore install-sh ltmain.sh Makefile.in missing .reviewboardrc
```
