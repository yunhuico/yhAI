FROM linkerrepository/mesos:2.0.0-1.0.1
MAINTAINER ychen <ychen@linkernetworks.com>

ENV MESOS_LOG_DIR /var/log/mesos
ENV MESOS_WORK_DIR /var/lib/mesos
ENV MESOS_REGISTRY replicated_log


ADD entrypoint.sh /
COPY static /usr/share/mesos/webui/master/static
RUN chmod +x /entrypoint.sh

EXPOSE 5050

VOLUME ["/var/lib/mesos"]

CMD ["/entrypoint.sh"]

