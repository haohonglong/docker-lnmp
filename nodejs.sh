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



#安装node.js
curl -o node.tar.xz https://nodejs.org/dist/v${NODE_VER}/node-v${NODE_VER}-linux-x64.tar.xz && tar -xvf node.tar.xz \
    && mv node-v${NODE_VER}-linux-x64 /usr/local/node \
    && sed -i "s/export PATH/PATH=\/usr\/local\/node\/bin:\$PATH\nexport PATH/" /etc/profile \
    && source /etc/profile && /usr/local/node/bin/npm install -g cnpm --registry=https://registry.npm.taobao.org \
    && rm -rf node.tar.gz


#安装mongodb
curl -o mongodb.tar.gz https://fastdl.mongodb.org/linux/mongodb-linux-x86_64-${MONGODB_VER}.tgz && tar -xvf mongodb.tar.gz \
    && mkdir -p ${MONGODB_DATA_DIR} && mv mongodb-linux-x86_64-${MONGODB_VER} /usr/local/mongodb \
    && sed -i "s/export PATH/PATH=\/usr\/local\/mongodb\/bin:\$PATH\nexport PATH/" /etc/profile  \
    && mkdir -p /var/log/mongodb \
    && echo -e "fork=false\njournal=true\nquiet=false\nlogpath=/var/log/mongodb/mongodb.log\nlogappend=true\nport=27017\ndbpath=${MONGODB_DATA}\nbind_ip=0.0.0.0" > /etc/mongod.conf \
    && rm -rf mongodb.tar.gz

