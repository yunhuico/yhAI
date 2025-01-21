From linkerrepository/linker_centos_java

MAINTAINER linker

WORKDIR /linker

#RUN yum -y update; yum clean all; \

ENV VERSION_MESOSDNS=0.5.2

ADD https://github.com/mesosphere/mesos-dns/releases/download/v${VERSION_MESOSDNS}/mesos-dns-v${VERSION_MESOSDNS}-linux-amd64 /linker/mesos-dns

RUN chmod 755 /linker/mesos-dns
RUN mkdir /linker/config

EXPOSE 53 8123

VOLUME ["/linker/config","/linker/log"]

CMD ["/linker/mesos-dns", "-config=/linker/config/config.json"]

