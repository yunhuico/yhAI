FROM centos:centos7
MAINTAINER ychen <ychen@linkernetworks.com>

ENV M_VERSION=1.4.2

RUN yum -y update
RUN rpm -Uvh http://repos.mesosphere.com/el/7/noarch/RPMS/mesosphere-el-repo-7-1.noarch.rpm
RUN yum -y install marathon-${M_VERSION} mesos-1.2.1
RUN yum -y install iproute.x86_64

ADD entrypoint.sh /
RUN chmod +x /entrypoint.sh

EXPOSE 8080

ENV MARATHON_LOGGING_LEVEL warn

CMD ["/entrypoint.sh"]
