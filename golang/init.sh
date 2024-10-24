#!/bin/bash
# 安装Go语言环境脚本
 
# 设置Go版本常量
GO_VERSION="1.22.1"
 
# 安装Go语言环境
echo "开始安装Go ${GO_VERSION} ..."

cd /tmp 
# 下载Go压缩包
wget "https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz"
 
# 创建Go目录结构

 
# 解压缩Go压缩包到/usr/local/go
sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
 
# 设置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPROXY=https://goproxy.cn' >> ~/.bashrc
source ~/.bashrc
 
# 验证Go版本
go version
 
# 清理安装文件
rm "go${GO_VERSION}.linux-amd64.tar.gz"
 
echo "Go ${GO_VERSION} 安装完成。"


