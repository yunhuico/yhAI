# syntax=docker/dockerfile:1.4
FROM alpine
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories \
    && apk update \
    && apk add --no-cache tzdata ca-certificates shared-mime-info \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone
WORKDIR /ultrafox
COPY  bin/ultrafox /bin/ultrafox
ENV GIN_MODE=release
EXPOSE 8010 8011
ENTRYPOINT ["/bin/ultrafox"]