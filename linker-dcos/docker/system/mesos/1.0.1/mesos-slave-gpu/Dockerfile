FROM linkerrepository/mesos:2.0.0-1.0.1

#LABEL com.nvidia.volumes.needed="nvidia_driver"

# install docker using default installation script
RUN curl -sSL https://s3-ap-southeast-1.amazonaws.com/docker-engine/v1.12.0/docker-install.sh | sh -

# instal dvdcli for docker/volume
RUN curl -sSL https://dl.bintray.com/emccode/dvdcli/install | sh -s stable

# install rexray and contiv for aws's ebs and ceph
#RUN curl -sSL https://dl.bintray.com/emccode/rexray/install | sh -s -- stable 0.3.3

EXPOSE 5051

#ENV MESOS_CONTAINERIZERS mesos
#ENV MESOS_LOG_DIR /var/log/mesos

ADD entrypoint.sh /
RUN chmod +x /entrypoint.sh

VOLUME ["/sys/fs/cgroup", "/tmp", "/opt/mesosphere", "/var/log/mesos", "/var/run/mesos"]

CMD ["/entrypoint.sh"]

# add nvidia libraries to system ld config
RUN echo "/usr/local/nvidia/lib" >> /etc/ld.so.conf.d/nvidia.conf && \
    echo "/usr/local/nvidia/lib64" >> /etc/ld.so.conf.d/nvidia.conf

ENV PATH /usr/local/nvidia/bin:/usr/local/cuda/bin:${PATH}
#ENV LD_LIBRARY_PATH /usr/local/nvidia/lib:/usr/local/nvidia/lib64
