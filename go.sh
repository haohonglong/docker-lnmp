#!/bin/bash

#go版本
GO_VER="1.10.3"
#安装go
cd /usr/src \
    && curl -o go.tar.gz https://studygolang.com/dl/golang/go${GO_VER}.linux-amd64.tar.gz -L \
    && mkdir /usr/local/go \
    && tar -xzvf go.tar.gz -C /usr/local/go  --strip-components 1 \
    && sed -i "s/export PATH/PATH=\/usr\/local\/go\/bin:\$HOME\/go\/bin:\$PATH\nGOPATH=\$HOME\/go\nexport GOPATH\nexport PATH/" /etc/profile \
    && sed -i 's/export PATH USER/export PATH USER GOPATH/' /etc/redis.conf \
    && rm -rf go.tar.gz



