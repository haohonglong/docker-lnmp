#!/bin/bash

#docker exec -it mysql mysqldump --opt --single-transaction --quick --lock-tables=false --all-databases > /vagrant_data/mysql/backup_all_databases.sql
docker pause mysql &&
sudo cp -r /home/vagrant/docker/dnmp/data/mysql/* /home/vagrant/projects/mysql/data &&
docker unpause mysql
echo "successful for backup"
