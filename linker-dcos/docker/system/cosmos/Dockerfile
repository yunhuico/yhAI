FROM java:openjdk-8-jdk

RUN wget https://bintray.com/artifact/download/sbt/debian/sbt-0.13.13.deb \
    && dpkg -i sbt-0.13.13.deb \
    && apt-get update \
    && apt-get install sbt 

RUN apt-get install git

ENV branch=v0.1.5-rc1

RUN git clone -b ${branch} https://github.com/xinxian0458/cosmos

RUN mkdir -p /linker /var/lib/cosmos

RUN cd cosmos \
    && sbt one-jar \
    && cd cosmos-server/target/scala-2.11/ \
    && ls \
    && mv cosmos-server_2.11-0.1.5-one-jar.jar cosmos.jar \
    && cp cosmos.jar /linker/

WORKDIR /linker

EXPOSE 7070

CMD ["java","-Xmx2G","-jar","/linker/cosmos.jar"]
