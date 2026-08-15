#!/bin/bash

# 当日委托功能完整演示
echo "=== Go当日委托服务完整演示 ==="
echo "服务运行在: http://localhost:8080"
echo ""

echo "1. ✅ 健康检查:"
curl -s http://localhost:8080/health
echo ""
echo ""

echo "2. 📊 服务状态:"
curl -s http://localhost:8080/api/orders/status | jq '.data'
echo ""
echo ""

echo "3. 📈 创建买入委托 (平安银行):"
curl -s -X POST http://localhost:8080/api/orders/create \
  -H "Content-Type: application/json" \
  -d '{
    "userid": "user_1001",
    "stock_code": "000001",
    "order_type": "1",
    "order_price": 10.8,
    "order_quantity": 500
  }' | jq '.data'
echo ""
echo ""

echo "4. 📉 创建卖出委托 (贵州茅台):"
curl -s -X POST http://localhost:8080/api/orders/create \
  -H "Content-Type: application/json" \
  -d '{
    "userid": "user_1002",
    "stock_code": "600519",
    "order_type": "2",
    "order_price": 1850.0,
    "order_quantity": 20
  }' | jq '.data'
echo ""
echo ""

echo "5. 🔍 查询用户1001的委托列表:"
curl -s "http://localhost:8080/api/orders/user/list?userid=user_1001" | jq '.data'
echo ""
echo ""

echo "6. 🔍 查询用户1002的委托列表:"
curl -s "http://localhost:8080/api/orders/user/list?userid=user_1002" | jq '.data'
echo ""
echo ""

echo "7. ⚡ 检查委托是否可成交:"
echo "   检查订单 ORDER_123456789 是否满足成交条件..."
curl -s -X POST http://localhost:8080/api/orders/match \
  -H "Content-Type: application/json" \
  -d '{"order_id": "ORDER_123456789"}' | jq '.data'
echo ""
echo ""

echo "8. ❌ 取消委托示例:"
echo "   取消订单 ORDER_987654321..."
curl -s -X POST http://localhost:8080/api/orders/cancel \
  -H "Content-Type: application/json" \
  -d '{"order_id": "ORDER_987654321"}' | jq '.data'
echo ""
echo ""

echo "9. 🏢 获取股票价格 (模拟服务):"
curl -s "http://localhost:8080/api/stocks/price"
echo ""
echo ""

echo "10. 🔄 委托检查服务状态:"
curl -s "http://localhost:8080/api/orders/check"
echo ""
echo ""

echo "11. 🧹 委托清理服务状态:"
curl -s "http://localhost:8080/api/orders/clean"
echo ""
echo ""

echo "=== 系统监控 ==="
echo "📊 查看服务日志: docker logs go-stock-dev --tail 20"
echo "🔄 服务状态: docker ps | grep go-stock-dev"
echo ""

echo "=== 技术架构说明 ==="
echo "• 前端: Go HTTP API (端口8080)"
echo "• 数据库: MySQL (通过GORM连接)"
echo "• 缓存: Redis (存储委托状态和股票价格)"
echo "• 后台服务:"
echo "  - 价格获取服务 (每分钟更新股票价格)"
echo "  - 委托监控服务 (每30秒检查委托状态)"
echo "  - 委托清理服务 (每小时清理过期委托)"
echo ""

echo "=== 当日委托功能特点 ==="
echo "✅ 创建当日委托 (买入/卖出)"
echo "✅ 查询委托详情"
echo "✅ 获取用户委托列表"
echo "✅ 取消未成交委托"
echo "✅ 检查委托成交条件"
echo "✅ 实时股票价格监控"
echo "✅ 委托状态自动更新"
echo "✅ 过期委托自动清理"
echo ""

echo "=== Docker容器信息 ==="
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep go-stock-dev