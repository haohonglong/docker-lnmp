package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"go-stock-services/internal/services"
	"go-stock-services/models"
)

// OrderAPI 委托订单API
type OrderAPI struct {
	orderService *services.OrderService
}

// NewOrderAPI 创建委托订单API实例
func NewOrderAPI(orderService *services.OrderService) *OrderAPI {
	return &OrderAPI{
		orderService: orderService,
	}
}

// CreateOrder 创建当日委托
// POST /api/orders/create
func (api *OrderAPI) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		UserID        string  `json:"userid"`
		StockID       string  `json:"stock_id"`
		SimulationID  string  `json:"simulation_id"`
		OrderType     string  `json:"order_type"`
		OrderPrice    float64 `json:"order_price"`
		OrderQuantity int     `json:"order_quantity"`
		StockCode     string  `json:"stock_code"`
		StockName     string  `json:"stock_name"`
		Exchange      string  `json:"exchange"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "解析请求失败: "+err.Error())
		return
	}

	order := &models.RedisOrder{
		UserID:        req.UserID,
		StockID:       req.StockID,
		SimulationID:  req.SimulationID,
		OrderType:     req.OrderType,
		OrderPrice:    models.FlexFloat(req.OrderPrice),
		OrderQuantity: models.FlexInt(req.OrderQuantity),
		StockCode:     req.StockCode,
		StockName:     req.StockName,
		Exchange:      req.Exchange,
	}

	orderID, err := api.orderService.CreateOrder(order)
	if err != nil {
		api.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	api.sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":  1,
		"message": "委托创建成功",
		"data": map[string]interface{}{
			"order_id": orderID,
		},
	})
}

// GetUserOrders 获取用户委托列表
// POST /simulation/getOrders  (兼容PHP前端POST form-data)
func (api *OrderAPI) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	// 兼容 GET 和 POST 两种方式
	r.ParseForm()
	userid := r.FormValue("userid")
	if userid == "" {
		userid = r.URL.Query().Get("userid")
	}
	if userid == "" {
		api.sendError(w, http.StatusBadRequest, "userid 参数是必须的")
		return
	}

	page, _ := strconv.Atoi(r.FormValue("page"))
	size, _ := strconv.Atoi(r.FormValue("size"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}

	orders, err := api.orderService.GetUserOrders(userid)
	if err != nil {
		api.sendError(w, http.StatusInternalServerError, "获取委托列表失败: "+err.Error())
		return
	}

	// 分页
	total := len(orders)
	offset := (page - 1) * size
	end := offset + size
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}

	pagedOrders := orders[offset:end]

	// 确保空slice序列化为[]而不是null
	if pagedOrders == nil {
		pagedOrders = make([]models.RedisOrder, 0)
	}

	api.sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":      1,
		"message":     "获取成功",
		"data":        pagedOrders,
		"total":       total,
		"page":        page,
		"size":        size,
		"total_pages": (total + size - 1) / size,
	})
}

// CancelOrder 取消委托
// POST /simulation/cancelOrder  (兼容PHP前端POST form-data)
func (api *OrderAPI) CancelOrder(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userid := r.FormValue("userid")
	stockID := r.FormValue("stock_id")
	orderID := r.FormValue("order_id")

	// 也尝试从JSON body读取
	if userid == "" || stockID == "" || orderID == "" {
		var req struct {
			UserID  string `json:"userid"`
			StockID string `json:"stock_id"`
			OrderID string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if userid == "" {
				userid = req.UserID
			}
			if stockID == "" {
				stockID = req.StockID
			}
			if orderID == "" {
				orderID = req.OrderID
			}
		}
	}

	if userid == "" || stockID == "" || orderID == "" {
		api.sendError(w, http.StatusBadRequest, "userid, stock_id, order_id 参数都是必须的")
		return
	}

	err := api.orderService.CancelOrder(userid, stockID, orderID)
	if err != nil {
		api.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	api.sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":  1,
		"message": "委托取消成功",
	})
}

// GetServiceStatus 获取服务状态
// GET /api/orders/status
func (api *OrderAPI) GetServiceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	api.sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":  1,
		"message": "服务运行正常",
		"data": map[string]interface{}{
			"service":   "当日委托服务",
			"version":   "1.0.0",
			"status":    "running",
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		},
	})
}

// CleanExpiredOrders 手动触发清理过期委托
// POST /api/orders/clean
func (api *OrderAPI) CleanExpiredOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	count, err := api.orderService.CleanExpiredOrders()
	if err != nil {
		api.sendError(w, http.StatusInternalServerError, "清理失败: "+err.Error())
		return
	}

	api.sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":        1,
		"message":       fmt.Sprintf("清理完成，共清理 %d 条过期委托", count),
		"cleaned_count": count,
	})
}

// HealthCheck 健康检查
func (api *OrderAPI) HealthCheck(w http.ResponseWriter, r *http.Request) {
	api.sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":    1,
		"message":   "服务健康",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	})
}

// sendJSON 发送JSON响应
func (api *OrderAPI) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[OrderAPI] JSON编码失败: %v", err)
	}
}

// sendError 发送错误响应
func (api *OrderAPI) sendError(w http.ResponseWriter, statusCode int, message string) {
	api.sendJSON(w, statusCode, map[string]interface{}{
		"status":  0,
		"message": message,
	})
}
