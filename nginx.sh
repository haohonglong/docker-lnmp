#!/bin/bash



#nginx版本
NGINX_VER=1.15.2


#安装nginx
cd /usr/src \
    && curl -o nginx.tar.gz http://nginx.org/download/nginx-${NGINX_VER}.tar.gz -L \
    && mkdir nginx && tar -xzvf nginx.tar.gz -C ./nginx --strip-components 1 \
    && cd nginx \
    && ./configure --prefix=/usr/local/nginx --conf-path=/etc/nginx/nginx.conf --error-log-path=/var/log/nginx/error.log \
        --http-log-path=/var/log/nginx/access.log --pid-path=/var/run/nginx.pid --lock-path=/var/lock/nginx.lock \
        --user=nginx --group=nginx --with-http_ssl_module --with-http_flv_module --with-http_stub_status_module \
        --with-http_gzip_static_module --http-client-body-temp-path=/tmp/nginx/client/ \
        --http-proxy-temp-path=/tmp/nginx/proxy/ \
        --http-fastcgi-temp-path=/tmp/nginx/fcgi/ \
        --with-pcre --with-http_dav_module \
    && make && sudo make install \
    && useradd nginx \
    && mkdir -p -m 777 /tmp/nginx \
    && echo "#!/bin/sh" > /etc/init.d/nginx \
    && echo "#description: Nginx web server." >> /etc/init.d/nginx \
    && echo -e "case \$1 in \n\
            restart): \n\
                /usr/local/nginx/sbin/nginx -s reload \n\
                ;; \n\
            stop): \n\
                /usr/local/nginx/sbin/nginx -s stop \n\
                ;; \n\
            *): \n\
                /usr/local/nginx/sbin/nginx \n\
                ;; \n\
        esac \n" >> /etc/init.d/nginx \
    && chmod +x /etc/init.d/nginx \
    #&& sed -i "3a daemon off;" /etc/nginx/nginx.conf \
    #&& sed -i "s/index  index.html index.htm;/index  index.php index.html index.htm;/" /etc/nginx/nginx.conf \
    #&& sed -i "s/# pass the PHP scripts to FastCGI server listening on 127.0.0.1:9000/location ~ \.php\$ { \nfastcgi_pass 127.0.0.1:9000;\nfastcgi_index  index.php;\nfastcgi_param  SCRIPT_FILENAME  \$document_root\$fastcgi_script_name;\ninclude fastcgi_params;\n }/" /etc/nginx/nginx.conf \
    && echo "<?php phpinfo()?>" > /usr/local/nginx/html/index.php \
    && rm -rf /etc/nginx && cp -rf /usr/src/etc/nginx /etc/nginx \
    && mkdir -m 777 -p /var/log/nginx \
    && rm -rf /usr/src/nginx.tar.gz && rm -rf /usr/src/nginx



