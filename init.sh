#!/bin/bash
# $1 container_id
# $2 image_id
if [ $1 ]; then 
    docker stop $1 && docker rm $1 
fi  

if [ $2 ]; then
    docker rmi $2
    docker build -t haohonglong/lnmp:1.0 .
fi  

docker run -h lam -p 80:80 -p 8080:8080 -p 3306:3306 -p 6379:6379 -p 27017:27017 --name lnmp -d \
-v /usr/local/var/nginx:/etc/nginx \
-v /usr/local/var/mysql:/data/mysql \
-v /usr/local/var/log:/var/log \
-v /usr/local/var/mongodb:/data/db \
-v /Users/long/sites:/usr/local/nginx/html haohonglong/lnmp:1.1
