FROM redis:2.8.19

RUN mkdir -p /var/lib/redis && \
    chown -R redis:redis /var/lib/redis

COPY redis.conf /etc/redis.conf

CMD ["redis-server", "/etc/redis.conf"]
