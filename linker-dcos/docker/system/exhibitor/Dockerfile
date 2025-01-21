FROM maven:3.3.9-jdk-8

# set up Environments
ENV \
	ZK_RELEASE="http://www.apache.org/dist/zookeeper/zookeeper-3.4.8/zookeeper-3.4.8.tar.gz" 

RUN \
	# install zookeeper
	curl -Lo /tmp/zookeeper.tgz $ZK_RELEASE \
	&& mkdir -p /opt/zookeeper/transactions /opt/zookeeper/snapshot \
	&& tar -xzf /tmp/zookeeper.tgz -C /opt/zookeeper --strip=1 \
	&& rm /tmp/zookeeper.tgz \

	# install exhibitor
	&& mkdir -p /opt/exhibitor 

# add the wrapper script to setup configs and exec exhibitor
env branch=master
RUN git clone -b $branch https://github.com/dcos/exhibitor /exhibitor\
  && cd /exhibitor && ./gradlew install \
  && ./gradlew -b exhibitor-standalone/src/main/resources/buildscripts/standalone/gradle/build.gradle shadowJar \
  && cp /exhibitor/exhibitor-standalone/src/main/resources/buildscripts/standalone/gradle/build/libs/exhibitor-1.5.6-SNAPSHOT-all.jar /opt/exhibitor/exhibitor.jar

ADD start_exhibitor.sh /opt/exhibitor/start_exhibitor.sh

USER root
WORKDIR /opt/exhibitor
EXPOSE 2181 2888 3888 8181

VOLUME ["/opt/zookeeper/snapshot", "/opt/zookeeper/transactions", "/opt/zookeeper/config"]

ENTRYPOINT ["bash", "-ex", "/opt/exhibitor/start_exhibitor.sh"]
