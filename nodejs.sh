#!/bin/bash


#node.js版本
NODE_VER="8.11.4"
#mongodb版本
MONGODB_VER="4.0.1"
#mongodb data目录
MONGODB_DATA_DIR="/data/mongodb"


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

