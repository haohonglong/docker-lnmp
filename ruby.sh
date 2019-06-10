#!/bin/bash
name=$1
name=${name:-'ruby-2.6.3'}
suffix=$2
suffix=${suffix:-'tar.gz'}
mkdir /ruby
cd /ruby
curl -o ${name}.${suffix} https://cache.ruby-lang.org/pub/ruby/2.6/${name}.${suffix}
if [ -e ${name}.${suffix} ];then
    tar -xvzf ${name}.${suffix}
    cd ${name} 
    ./configure
    make
    make install
    cd /
    rm -rf /ruby 
else
    echo nothing
fi 
