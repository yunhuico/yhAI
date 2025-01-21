FROM linkerrepository/mesos:2.0.0-1.0.1
MAINTAINER ychen <ychen@linkernetworks.com>

# install docker using default installation script
RUN curl -sSL https://s3-ap-southeast-1.amazonaws.com/docker-engine/v1.12.0/docker-install.sh | sh -

# instal dvdcli for docker/volume
RUN curl -sSL https://dl.bintray.com/emccode/dvdcli/install | sh -s stable

# install rexray and contiv for aws's ebs and ceph
#RUN curl -sSL https://dl.bintray.com/emccode/rexray/install | sh -s -- stable 0.3.3

# backup mesos libraries
RUN mv /usr/lib/libmesos-1.0.1.so /usr/lib/libmesos-1.0.1.so.old && \
    mv 	/usr/lib/libmesos.la /usr/lib/libmesos.la.old

# TODO: add new libraries and modules
COPY libmesos-1.0.1.so /usr/lib/libmesos-1.0.1.so
COPY libmesos.la /usr/lib/libmesos.la
COPY libpostlaunchdockerhook-1.0.1.so /usr/lib/mesos/modules/libpostlaunchdockerhook-1.0.1.so
COPY libpostlaunchdockerhook.la /usr/lib/mesos/modules/libpostlaunchdockerhook.la
RUN ln -s /usr/lib/mesos/modules/libpostlaunchdockerhook-1.0.1.so /usr/lib/mesos/modules/libpostlaunchdockerhook.so

COPY post_launch_docker_hook.json /opt/post_launch_docker_hook.json
COPY linkerconfig /opt/linkerconfig
# TODO: RUN chmod +x /opt/linkerconfig

ENV MESOS_MODULES file:///opt/post_launch_docker_hook.json
ENV MESOS_HOOKS org_apache_mesos_PostLaunchDockerHook

RUN apt-get install libcurl3-nss -y

EXPOSE 5051

ENV MESOS_CONTAINERIZERS docker,mesos
ENV MESOS_LOG_DIR /var/log/mesos

ADD entrypoint.sh /
RUN chmod +x /entrypoint.sh

VOLUME ["/sys/fs/cgroup", "/var/run/docker.sock", "/tmp", "/opt/mesosphere", "/var/log/mesos"]

CMD ["/entrypoint.sh"]

