FROM ubuntu:15.10

RUN echo "deb http://archive.ubuntu.com/ubuntu wily main restricted universe multiverse" > /etc/apt/sources.list
RUN echo "deb http://archive.ubuntu.com/ubuntu wily-updates main restricted universe multiverse" >> /etc/apt/sources.list

RUN apt-get -y update && apt-get -y install kafkacat

ENTRYPOINT ["kafkacat"]
