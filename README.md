docker run -h lam -p 80:80 -p 8080:8080 -p 9001:9001 -p 3306:3306 -p 6379:6379 -p 27017:27017 --name lnmp -d \
  -v /usr/local/var/nginx:/etc/nginx \
  -v /usr/local/var/mysql:/data/mysql \
  -v /usr/local/var/log:/var/log \
  -v /Users/long/sites:/usr/local/nginx/html haohonglong/lnmp:1.0


export PATH=$PATH:/usr/local/nginx/sbin


#### the grant of mysql
    drop database if exists litemall;
    drop user if exists 'litemall'@'%';
    -- 支持emoji：需要mysql数据库参数： character_set_server=utf8mb4
    create database litemall default character set utf8mb4 collate utf8mb4_unicode_ci;
    use litemall;
    create user 'litemall'@'%' identified by 'litemall123456';
    grant all privileges on litemall.* to 'litemall'@'%';
    flush privileges;

    CREATE USER 'lam'@'%' IDENTIFIED BY '123456';
    GRANT ALL PRIVILEGES ON *.* TO 'lam'@'%' IDENTIFIED BY '123456' WITH GRANT OPTION;
    FLUSH   PRIVILEGES;


#### No space left on devic have been solved in linux
    # Remove stopped containers, unused networks, dangling images, and build cache
    docker system prune -af --volumes

    # checking the size of folders of files
    sudo du -ah -x --max-depth=1
    
    du -sh ./*

    df -h
