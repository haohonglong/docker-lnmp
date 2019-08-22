#!/bin/bash



#root用户密码
ROOT_PASSWORD=123456
#php版本,因为php版本间配置文件模板不相同，此处的版本号只能为大于7.0以上版本
PHP_VER=7.2.8
#nginx版本
NGINX_VER=1.15.2
#mysql版本
MYSQL_VER=5.7.23
#redis版本
REDIS_VER=4.0.11
#redis密码
REDIS_PASS=123456
#phpmyadmin版本
PHPMYADMIN_VER=4.7.6
#mysql data目录
MYSQL_DATA_DIR="/data/mysql"
#mysql pid目录
MYSQL_PID_DIR="/var/run/mysql"
#mysql log目录
MYSQL_LOG_DIR="/var/log/mysql"
#mysql sock目录
MYSQL_SOCK_DIR="/var/lib/mysql"
#xhprof版本
XHPROF_VER="1.2"
#hiredis版本
HIREDIS_VER="0.13.3"
#swoole版本
SWOOLE_VER="4.0.4"
#go版本
GO_VER="1.10.3"
#node.js版本
NODE_VER="8.11.4"
#mongodb版本
MONGODB_VER="4.0.1"
#mongodb data目录
MONGODB_DATA_DIR="/data/mongodb"
#java版本
JDK_VER="1.8"
#maven版本
MAVEN_VER="3.6.0"


#安装go
cd /usr/src \
    && curl -o go.tar.gz https://studygolang.com/dl/golang/go${GO_VER}.linux-amd64.tar.gz -L \
    && mkdir /usr/local/go \
    && tar -xzvf go.tar.gz -C /usr/local/go  --strip-components 1 \
    && sed -i "s/export PATH/PATH=\/usr\/local\/go\/bin:\$HOME\/go\/bin:\$PATH\nGOPATH=\$HOME\/go\nexport GOPATH\nexport PATH/" /etc/profile \
    && sed -i 's/export PATH USER/export PATH USER GOPATH/' /etc/redis.conf \
    && rm -rf go.tar.gz



