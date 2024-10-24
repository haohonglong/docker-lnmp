#!/bin/bash

nvm install 20

# install the yarn
corepack enable
yarn -v

npm config set registry https://registry.npmmirror.com
npm config get registry
