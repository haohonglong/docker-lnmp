package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-stock-services/config"
	"go-stock-services/database"
	"go-stock-services/internal/api"
	"go-stock-services/internal/services"
)

func main() {
	// 加载配置
	cfg := config.LoadConfig()

	// 初始化数据库连接
	if err := database.InitDB(cfg); err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}

	// 初始化Redis连接
	if err := database.InitRedis(cfg); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// 获取数据库和Redis连接
	db := database.GetDB()
	redisClient := database.GetRedis()

	// 初始化服务
	orderService := services.NewOrderService(db, redisClient)
	priceFetcher := services.NewPriceFetcher(db, redisClient, services.PriceFetcherConfig{
		FetchInterval:      cfg.PriceUpdateInterval,
		OrderCheckInterval: cfg.OrderCheckInterval,
		APIBaseURL:         cfg.StockAPIURL,
		PHPAPIURL:          cfg.PHPAPIURL,
	})
	orderCleaner := services.NewOrderCleaner(db, redisClient)

	// 初始化API
	orderAPI := api.NewOrderAPI(orderService)
	wsLogAPI := api.NewWSLogAPI(redisClient)

	// 启动Redis日志订阅
	wsLogAPI.StartSubscriber()

	// 启动HTTP服务
	router := setupRouter(orderAPI, wsLogAPI)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      corsMiddleware(router),
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
	}

	// 启动服务
	go func() {
		log.Printf("Starting HTTP server on port %d", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// 启动后台服务
	go priceFetcher.Start()
	go orderCleaner.Start()

	log.Printf("Go Stock Services 启动完成 - 服务地址: http://localhost:%d", cfg.HTTPPort)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// 停止所有服务
	priceFetcher.Stop()
	orderCleaner.Stop()

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupRouter(orderAPI *api.OrderAPI, wsLogAPI *api.WSLogAPI) *http.ServeMux {
	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":1,"message":"OK"}`))
	})

	// 委托订单API（与PHP接口兼容）
	mux.HandleFunc("/api/orders/create", orderAPI.CreateOrder)
	mux.HandleFunc("/api/orders/user/list", orderAPI.GetUserOrders)
	mux.HandleFunc("/api/orders/cancel", orderAPI.CancelOrder)
	mux.HandleFunc("/api/orders/clean", orderAPI.CleanExpiredOrders)
	mux.HandleFunc("/api/orders/status", orderAPI.GetServiceStatus)

	// WebSocket日志推送
	mux.HandleFunc("/ws/logs", wsLogAPI.HandleWebSocket)

	// 向后兼容的旧路由
	mux.HandleFunc("/simulation/getOrders", orderAPI.GetUserOrders)
	mux.HandleFunc("/simulation/cancelOrder", orderAPI.CancelOrder)

	return mux
}

// corsMiddleware CORS中间件，允许前端跨域访问
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
