#!/bin/sh
# 清空Redis DB0中所有委托数据（每天开盘前执行）
docker exec redis redis-cli -n 0 -a '123456' EVAL "local keys = redis.call('KEYS', 'stock_orders:*:*') if #keys > 0 then for _, key in ipairs(keys) do redis.call('DEL', key) end return #keys end return 0" 0