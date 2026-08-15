#!/bin/bash

# Go Stock Services API 测试脚本
# 用法: ./test_order_api.sh [base_url]

BASE_URL="${1:-http://localhost:8080}"
echo "=== Go Stock Services API 测试 ==="
echo "服务地址: $BASE_URL"
echo ""

# 1. 健康检查
echo "1. 健康检查:"
curl -s "$BASE_URL/health" | python3 -m json.tool 2>/dev/null || curl -s "$BASE_URL/health"
echo -e "\n"

# 2. 服务状态
echo "2. 服务状态:"
curl -s "$BASE_URL/api/orders/status" | python3 -m json.tool 2>/dev/null || curl -s "$BASE_URL/api/orders/status"
echo -e "\n"

# 3. 创建买入委托
echo "3. 创建买入委托 (平安银行 10.50 买入100股):"
RESULT=$(curl -s -X POST "$BASE_URL/api/orders/create" \
  -H "Content-Type: application/json" \
  -d '{
    "userid": "1",
    "stock_id": "1",
    "simulation_id": "test_sim_001",
    "stock_code": "000001",
    "stock_name": "平安银行",
    "exchange": "SZ",
    "order_type": "1",
    "order_price": 10.50,
    "order_quantity": 100
  }')
echo "$RESULT" | python3 -m json.tool 2>/dev/null || echo "$RESULT"
ORDER_ID=$(echo "$RESULT" | grep -o '"order_id":"[^"]*"' | cut -d'"' -f4)
echo -e "\n"

# 4. 创建卖出委托
echo "4. 创建卖出委托 (贵州茅台 1800.00 卖出10股):"
curl -s -X POST "$BASE_URL/api/orders/create" \
  -H "Content-Type: application/json" \
  -d '{
    "userid": "1",
    "stock_id": "2",
    "simulation_id": "test_sim_002",
    "stock_code": "600519",
    "stock_name": "贵州茅台",
    "exchange": "SH",
    "order_type": "2",
    "order_price": 1800.00,
    "order_quantity": 10
  }' | python3 -m json.tool 2>/dev/null || echo "$RESULT"
echo -e "\n"

# 5. 获取用户委托列表
echo "5. 获取用户1的委托列表:"
curl -s "$BASE_URL/api/orders/user/list?userid=1" | python3 -m json.tool 2>/dev/null || curl -s "$BASE_URL/api/orders/user/list?userid=1"
echo -e "\n"

# 6. 取消委托
if [ -n "$ORDER_ID" ]; then
  echo "6. 取消委托 (order_id=$ORDER_ID):"
  curl -s -X POST "$BASE_URL/api/orders/cancel" \
    -H "Content-Type: application/json" \
    -d "{\"userid\": \"1\", \"stock_id\": \"1\", \"order_id\": \"$ORDER_ID\"}" | python3 -m json.tool 2>/dev/null || echo "$RESULT"
  echo -e "\n"
else
  echo "6. 跳过取消委托（未获取到order_id）"
  echo ""
fi

# 7. 再次获取用户委托列表（确认取消成功）
echo "7. 再次获取用户1的委托列表（确认取消成功）:"
curl -s "$BASE_URL/api/orders/user/list?userid=1" | python3 -m json.tool 2>/dev/null || curl -s "$BASE_URL/api/orders/user/list?userid=1"
echo -e "\n"

# 8. 手动触发清理过期委托
echo "8. 手动触发清理过期委托:"
curl -s -X POST "$BASE_URL/api/orders/clean" | python3 -m json.tool 2>/dev/null || curl -s -X POST "$BASE_URL/api/orders/clean"
echo -e "\n"

echo "=== 测试完成 ==="
