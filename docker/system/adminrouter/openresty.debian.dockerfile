FROM debian:wheezy

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
    curl cron perl make build-essential procps vim  \
    libreadline-dev libncurses5-dev libpcre3-dev libssl-dev \
 && rm -rf /var/lib/apt/lists/* \
 && update-rc.d cron defaults

ENV OPENRESTY_VERSION 1.9.7.3
ENV OPENRESTY_PREFIX /opt/openresty
ENV NGINX_PREFIX /opt/openresty/nginx
ENV VAR_PREFIX /var/nginx

# NginX prefix is automatically set by OpenResty to $OPENRESTY_PREFIX/nginx
# look for $ngx_prefix in https://github.com/openresty/ngx_openresty/blob/master/util/configure

RUN cd /root \
 && echo "==> Downloading OpenResty..." \
 && curl -sSL http://openresty.org/download/openresty-${OPENRESTY_VERSION}.tar.gz | tar -xvz \
 && echo "==> Configuring OpenResty..." \
 && cd openresty-* \
 && readonly NPROC=$(grep -c ^processor /proc/cpuinfo 2>/dev/null || 1) \
 && echo "using upto $NPROC threads" \
 && ./configure \
    --prefix=$OPENRESTY_PREFIX \
    --http-client-body-temp-path=$VAR_PREFIX/client_body_temp \
    --http-proxy-temp-path=$VAR_PREFIX/proxy_temp \
    --http-log-path=$VAR_PREFIX/access.log \
    --error-log-path=$VAR_PREFIX/error.log \
    --pid-path=$VAR_PREFIX/nginx.pid \
    --lock-path=$VAR_PREFIX/nginx.lock \
    --with-luajit \
    --with-pcre-jit \
    --with-ipv6 \
    --with-http_ssl_module \
    --without-http_ssi_module \
    --without-http_userid_module \
    --without-http_uwsgi_module \
    --without-http_scgi_module \
    -j${NPROC} \
 && echo "==> Building OpenResty..." \
 && make -j${NPROC} \
 && echo "==> Installing OpenResty..." \
 && make install \
 && echo "==> Finishing..." \
 && ln -sf $NGINX_PREFIX/sbin/nginx /usr/local/bin/nginx \
 && ln -sf $NGINX_PREFIX/sbin/nginx /usr/local/bin/openresty \
 && ln -sf $OPENRESTY_PREFIX/bin/resty /usr/local/bin/resty \
 && ln -sf $OPENRESTY_PREFIX/luajit/bin/luajit-* $OPENRESTY_PREFIX/luajit/bin/lua \
 && ln -sf $OPENRESTY_PREFIX/luajit/bin/luajit-* /usr/local/bin/lua \
 && rm -rf /root/ngx_openresty*

WORKDIR $NGINX_PREFIX/

ONBUILD RUN rm -rf conf/* html/*
ONBUILD COPY nginx $NGINX_PREFIX/

CMD ["nginx", "-g", "daemon off; error_log /dev/stderr info;"]
