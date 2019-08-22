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



#安装Java JDK
cd /usr/src \
    && curl -o jdk.tar.gz http://d.feehi.com/jdk-${JDK_VER} -L \
    && mkdir -p /usr/local/java && tar -xzvf jdk.tar.gz -C /usr/local/java --strip-components 1 \
    && sed -i "s/export PATH/PATH=\/usr\/local\/java\/bin:\$PATH\nJAVA_HOME=\/usr\/local\/java\nJRE_HOME=\/usr\/local\/java\/jre\nCLASSPATH=\.:\$JAVA_HOME\/lib:\$JRE_HOME\/lib\nexport JAVA_HOME JRE_HOME CLASSPATH\nexport PATH/" /etc/profile \
    && rm -rf jdk.tar.gz


#安装maven
cd /usr/src && curl -o maven.tar.gz http://mirrors.tuna.tsinghua.edu.cn/apache/maven/maven-3/${MAVEN_VER}/binaries/apache-maven-${MAVEN_VER}-bin.tar.gz -L \
    && tar -xvf maven.tar.gz && mv apache-maven-${MAVEN_VER} /usr/local/maven \
    && sed -i "s/<mirrors>/<mirrors><mirror><id>nexus-aliyun<\/id><mirrorOf>central<\/mirrorOf><name>Nexus aliyun<\/name><url>http:\/\/maven.aliyun.com\/nexus\/content\/groups\/public<\/url><\/mirror>/" /usr/local/maven/conf/settings.xml \
    && sed -i "s/export PATH/MAVEN_HOME=\/usr\/local\/maven \nexport MAVEN_HOME\nPATH=\/usr\/local\/maven\/bin:\$PATH\nexport PATH/" /etc/profile \
    && rm -rf maven.tar.gz




