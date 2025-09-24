#!/bin/bash

# 更新系统
apt-get update
apt-get upgrade -y

# 安装必要的包
apt-get install -y apt-transport-https ca-certificates curl software-properties-common

# 使用国内Docker镜像源（阿里云）
# 添加Docker的GPG密钥（使用阿里云的镜像）
curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg | sudo apt-key add -

# 添加Docker仓库（使用阿里云镜像）
sudo add-apt-repository "deb [arch=amd64] https://mirrors.aliyun.com/docker-ce/linux/ubuntu $(lsb_release -cs) stable"

# 更新包索引
apt-get update

# 安装Docker
apt-get install -y docker-ce

# 将当前用户添加到docker组（避免每次使用sudo）
usermod -aG docker vagrant

# 启用Docker服务
systemctl enable docker
systemctl start docker

# 安装Docker Compose
curl -L "https://github.com/docker/compose/releases/download/1.29.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose