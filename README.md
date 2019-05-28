docker run -h lam -p 80:80 -p 8080:8080 -p 9001:9001 -p 3306:3306 -p 6379:6379 -p 27017:27017 --name lnmp -d \
  -v /usr/local/var/nginx:/etc/nginx \
  -v /usr/local/var/mysql:/data/mysql \
  -v /usr/local/var/log:/var/log \
  -v /Users/long/sites:/usr/local/nginx/html haohonglong/lnmp:1.0


export PATH=$PATH:/usr/local/nginx/sbin


#### the grant of mysql
    GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' IDENTIFIED BY '123456' WITH GRANT OPTION;
    FLUSH   PRIVILEGES;
