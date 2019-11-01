ARG OS_VER=latest

FROM registry.cn-hangzhou.aliyuncs.com/liufee/feehi:$OS_VER
MAINTAINER lam github.com/haohonglong/lam2

#COPY ./ruby.sh /

RUN yum install clang

#install xdebug
RUN cd /usr/src \
    && mkdir xdebug \
    && cd xdebug \
    && curl -o xdebug.tgz http://xdebug.org/files/xdebug-2.7.2.tgz -L \
    && tar -xvzf xdebug.tgz \
    && cd xdebug-2.7.2 \
    && /usr/local/php/bin/phpize \
    && ./configure \
    && make \
    && cp modules/xdebug.so /usr/local/php/lib/php/extensions/no-debug-zts-20170718 \
    && echo "zend_extension = /usr/local/php/lib/php/extensions/no-debug-zts-20170718/xdebug.so" >> /etc/php/php.ini \
    && echo "[XDebug]" >> /etc/php/php.ini \
    && echo "xdebug.remote_enable = 1" >> /etc/php/php.ini \
    && echo "xdebug.idekey = phpstorm" >> /etc/php/php.ini \
    && echo "xdebug.remote_autostart = 0" >> /etc/php/php.ini \
    && echo "xdebug.remote_port = 9001" >> /etc/php/php.ini \
    && echo "xdebug.remote_host = 127.0.0.1" >> /etc/php/php.ini \
    && echo "xdebug.remote_handler = dbgp" >> /etc/php/php.ini \
    && echo "xdebug.remote_log = /var/log/xdebug_remote.log" >> /etc/php/php.ini \
    && rm -rf /usr/src/xdebug
    
#install yaf

#install ruby
#RUN chmod 755 /ruby.sh



RUN ln -s /usr/local/nginx/html /www


EXPOSE 8080



