ARG OS_VER=latest


FROM centos:$OS_VER
MAINTAINER lam github.com/haohonglong/docker-lnmp


#root用户密码
ARG ROOT_PASSWORD=123456
#php版本,因为php版本间配置文件模板不相同，此处的版本号只能为大于7.0以上版本
ARG PHP_VER=7.2.8
#nginx版本
ARG NGINX_VER=1.15.2
#xhprof版本
ARG XHPROF_VER=1.2
#swoole版本
ARG SWOOLE_VER=4.0.4


#映射配置文件
ADD ./etc /usr/src/etc


#基础环境配置
RUN yum install vim wget git net-tools -y \
    && yum install epel-release -y \
    && yum update -y \
    && yum -y install pcre pcre-devel zlib zlib-devel openssl openssl-devel libxml2 libxml2-devel libjpeg libjpeg-devel \
        libpng libpng-devel curl curl-devel libicu libicu-devel libmcrypt  libmcrypt-devel freetype freetype-devel \
        libmcrypt libmcrypt-devel autoconf gcc-c++ gcc make automake cmake ncurses-devel bison bison-devel\
    && yum install vixie-cron crontabs -y \
    && yum install python-setuptools -y \
    && easy_install supervisor \
    && yum install openssh-server -y \
    && echo PermitRootLogin  yes >> /etc/ssh/sshd_config \
    && echo PasswordAuthentication yes >> /etc/ssh/sshd_config \
    && echo RSAAuthentication yes >> etc/ssh/sshd_config \
    && sed -i "s/UseDNS yes/UseDNS no/" /etc/ssh/sshd_config \
    && echo "root:$ROOT_PASSWORD" | chpasswd \
    && ssh-keygen -t dsa -f /etc/ssh/ssh_host_rsa_key \
    && ssh-keygen -t rsa -f /etc/ssh/ssh_host_ecdsa_key \
    && ssh-keygen -t rsa -f /etc/ssh/ssh_host_ed25519_key \
    && yum clean all && rm -rf /var/cache/yum/*


#安装php
RUN cd /usr/src \
    && curl -o php.tar.gz http://php.net/get/php-${PHP_VER}.tar.gz/from/this/mirror -L \
    && mkdir php \
    && tar -xzvf php.tar.gz -C ./php --strip-components 1 \
    && cd php \
    && ./configure --prefix=/usr/local/php --with-config-file-path=/etc/php --enable-soap --enable-mbstring=all \
        --enable-sockets --enable-fpm --with-gd --with-freetype-dir=/usr/include/freetype2/freetype \
        --with-jpeg-dir=/usr/lib64 --with-zlib --with-iconv --enable-libxml --enable-xml  --enable-intl \
        --enable-zip --enable-pcntl --enable-bcmath --enable-maintainer-zts --with-curl --with-mcrypt --with-openssl \
        --with-mysqli=mysqlnd --with-pdo-mysql=mysqlnd \
    && make \
    && make install \
    && mkdir /etc/php \
    && cp /usr/src/php/php.ini-development /etc/php/php.ini \
    && echo $MYSQL_SOCK_DIR > /tmp/temp_mysql_sock_dir.txt && temp_mysql_sock_dir=$(sed "s/\//\\\\\//g" /tmp/temp_mysql_sock_dir.txt) && rm -rf /tmp/temp_mysql_sock_dir.txt \
    && sed -i "s/mysqli.default_socket =/mysqli.default_socket = ${temp_mysql_sock_dir}\/mysql.sock/" /etc/php/php.ini \
    && sed -i "s/pdo_mysql.default_socket=/pdo_mysql.default_socket = ${temp_mysql_sock_dir}\/mysql.sock/" /etc/php/php.ini \
    && cp /usr/src/php/sapi/fpm/init.d.php-fpm /etc/init.d/php-fpm \
    && chmod +x /etc/init.d/php-fpm \
    && cd /usr/local/php/etc \
    && cp php-fpm.conf.default php-fpm.conf \
    && sed -i "s/;daemonize = yes/daemonize = no/" php-fpm.conf \
    && cp ./php-fpm.d/www.conf.default ./php-fpm.d/www.conf \
    && sed -i "s/export PATH/PATH=\/usr\/local\/php\/bin:\$PATH\nexport PATH/" /etc/profile \
    && sed -i "s/export PATH/PATH=\/etc\/init.d:\$PATH\nexport PATH/" /etc/profile \
    && rm -rf /usr/src/php.tar.gz && rm -rf /usr/src/php \
    #php redis扩展
    && /usr/local/php/bin/pecl install redis && echo "extension=redis.so" >> /etc/php/php.ini \
    #php swoole扩展
    && cd /usr/src && curl -o hiredis.tar.gz https://github.com/redis/hiredis/archive/v${HIREDIS_VER}.tar.gz -L && mkdir hiredis && tar -xzvf hiredis.tar.gz -C ./hiredis --strip-components 1 \
    && cd hiredis && make && make install && mkdir /usr/lib/hiredis && cp libhiredis.so /usr/lib/hiredis && mkdir /usr/include/hiredis && cp hiredis.h /usr/include/hiredis \
    && echo '/usr/local/lib' >> /etc/ld.so.conf && ldconfig \
    && /usr/local/php/bin/pecl download swoole-${SWOOLE_VER} && tar -zxvf swoole-${SWOOLE_VER}.tgz && cd swoole-${SWOOLE_VER} \
    && /usr/local/php/bin/phpize && ./configure --with-php-config=/usr/local/php/bin/php-config --enable-openssl --enable-async-redis && make && make install \
    && echo "extension=swoole.so" >> /etc/php/php.ini && rm -rf /usr/src/hiredis.tar.gz && rm -rf /usr/src/hiredis && rm -rf swoole-${SWOOLE_VER}.tgz && rm -rf swoole-${SWOOLE_VER} \
    #php xhprof扩展
    && cd /usr/src \
    && curl -o xhprof.tar.gz https://github.com/longxinH/xhprof/archive/v${XHPROF_VER}.tar.gz -L \
    && tar -xvf xhprof.tar.gz \
    && cd xhprof-${XHPROF_VER}/extension \
    && /usr/local/php/bin/phpize \
    && ./configure --with-php-config=/usr/local/php/bin/php-config --enable-xhprof && make && make install \
    && mkdir -p -m 777 /tmp/xhprof \
    && echo -e "[xhprof]\nextension = xhprof.so\nxhprof.output_dir = /tmp/xhprof" >> /etc/php/php.ini \
    && mkdir /var/tools \
    && cd /usr/src/xhprof-${XHPROF_VER} \
    && mv xhprof_html /var/tools/ \
    && mv xhprof_lib /usr/local/php/lib/php \
    && sed -i "s/dirname(__FILE__) . '\/..\/xhprof_lib'/'xhprof_lib'/" /var/tools/xhprof_html/index.php \
    && sed -i "s/dirname(__FILE__) . '\/..\/xhprof_lib'/'xhprof_lib'/" /var/tools/xhprof_html/callgraph.php \
    && sed -i "s/dirname(__FILE__) . '\/..\/xhprof_lib'/'xhprof_lib'/" /var/tools/xhprof_html/typeahead.php \
    && rm -rf /usr/src/xhprof-${XHPROF_VER} && rm -rf /usr/src/xhprof.tar.gz

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


#环境变量设置
ENV PATH /usr/local/php/bin:/etc/init.d:$PATH


#服务器基础设置
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo 'Asia/Shanghai' > /etc/timezonesource \
    && source /etc/profile \
    &&echo [supervisord] > /etc/supervisord.conf \
    && echo nodaemon=true >> /etc/supervisord.conf \
    && echo user=root >> /etc/supervisord.conf \
    \
    && echo [program:sshd] >> /etc/supervisord.conf \
    && echo command=/usr/sbin/sshd -D >> /etc/supervisord.conf \
    \
    && echo [program:php-fpm] >> /etc/supervisord.conf \
    && echo command=/usr/local/php/sbin/php-fpm >> /etc/supervisord.conf 
    

RUN ln -s /usr/local/nginx/html /www

EXPOSE 80 


CMD ["/usr/bin/supervisord"]
