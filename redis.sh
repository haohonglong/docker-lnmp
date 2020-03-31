#!/bin/bash



#redis版本
REDIS_VER=4.0.11
#redis密码
REDIS_PASS=123456

#安装redis server
cd /usr/src \
    && curl -o redis.tar.gz http://download.redis.io/releases/redis-${REDIS_VER}.tar.gz -L \
    && mkdir redis \
    && tar -xzvf redis.tar.gz -C ./redis --strip-components 1 \
    && cd redis \
    && make \
    && sudo make install \
    && mkdir -p /usr/local/redis/bin \
    && cp ./src/redis-server /usr/local/redis/bin/ \
    && cp ./src/redis-cli /usr/local/redis/bin/ \
    && cp ./src/redis-benchmark /usr/local/redis/bin/ \
    && cp ./redis.conf /etc/redis.conf \
    && sed -i 's/bind 127.0.0.1/bind 0.0.0.0/' /etc/redis.conf \
    && sed -i "s/# requirepass foobared/requirepass ${REDIS_PASS}/" /etc/redis.conf \
    && echo -e "# description: Redis server. \n\
         case \$1 in \n\
            restart): \n\
                /usr/local/redis/bin/redis-cli -h 127.0.0.1 -p 6379 -a 123456 shutdown \n\
                /usr/local/redis/bin/redis-server /etc/redis.conf \n\
                ;; \n\
            stop): \n\
                /usr/local/redis/bin/redis-cli -h 127.0.0.1 -p 6379 -a 123456 shutdown \n\
                ;; \n\
            *): \n\
                /usr/local/redis/bin/redis-server /etc/redis.conf \n\
         esac" > /etc/init.d/redis \
    && chmod +x /etc/init.d/redis \
    && sed -i "s/export PATH/PATH=\/usr\/local\/redis\/bin:\$PATH\nexport PATH/" /etc/profile \
    && rm -rf /usr/src/redis.tar.gz && rm -rf /usr/src/redis



