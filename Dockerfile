ARG OS_VER=latest

FROM registry.cn-hangzhou.aliyuncs.com/liufee/feehi:$OS_VER
MAINTAINER lam github.com/haohonglong/lam2

#install xdebug
RUN cd /usr/src \
    && mkdir xdebug \
    && cd xdebug \
    && curl -o xdebug.tgz http://xdebug.org/files/xdebug-2.7.2.tgz -L \
    && tar -xvzf xdebug.tgz \
    && cd xdebug-2.7.2 \
    && phpize \
    && ./configure \
    && make \
    && cp modules/xdebug.so /usr/local/php/lib/php/extensions/no-debug-zts-20170718 \
    && echo "zend_extension = /usr/local/php/lib/php/extensions/no-debug-zts-20170718/xdebug.so" >> /etc/php/php.ini \
    && echo "xdebug.remote_port = 9001" >> /etc/php/php.ini \
    && rm -rf /usr/src/xdebug


 RUN ln -s /usr/local/nginx/html /www


EXPOSE 8080 9001 9002


