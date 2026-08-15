package api

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"go-stock-services/internal/pubsub"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源（CORS）
	},
}

// WSLogAPI WebSocket日志推送API
type WSLogAPI struct {
	redis      *redis.Client
	ctx        context.Context
	clients    map[*websocket.Conn]bool
	clientsMux sync.Mutex
}

// NewWSLogAPI 创建WebSocket日志API实例
func NewWSLogAPI(redisClient *redis.Client) *WSLogAPI {
	return &WSLogAPI{
		redis:   redisClient,
		ctx:     context.Background(),
		clients: make(map[*websocket.Conn]bool),
	}
}

// HandleWebSocket 处理WebSocket连接
func (ws *WSLogAPI) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WSLogAPI] WebSocket升级失败: %v", err)
		return
	}
	defer conn.Close()

	// 注册客户端
	ws.clientsMux.Lock()
	ws.clients[conn] = true
	ws.clientsMux.Unlock()

	log.Printf("[WSLogAPI] 新的WebSocket连接, 当前连接数: %d", ws.getClientCount())

	// 发送最近一批日志（连接时通知客户端已连接）
	conn.WriteJSON(pubsub.LogMessage{
		Type:      "connected",
		Message:   "已连接到股票日志服务",
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	})

	// 保持连接，读取客户端消息（心跳等）
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			ws.clientsMux.Lock()
			delete(ws.clients, conn)
			ws.clientsMux.Unlock()
			log.Printf("[WSLogAPI] WebSocket连接断开, 当前连接数: %d", ws.getClientCount())
			break
		}
	}
}

// getClientCount 获取当前连接数
func (ws *WSLogAPI) getClientCount() int {
	return len(ws.clients)
}

// StartSubscriber 启动Redis订阅，监听日志频道
func (ws *WSLogAPI) StartSubscriber() {
	go func() {
		for {
			sub := ws.redis.Subscribe(ws.ctx, "stock:log")
			ch := sub.Channel()

			log.Println("[WSLogAPI] 已订阅 stock:log 频道")

			for msg := range ch {
				ws.broadcast(msg.Payload)
			}

			// 如果订阅断开，等待重连
			log.Println("[WSLogAPI] Redis订阅断开，5秒后重连...")
			time.Sleep(5 * time.Second)
		}
	}()
}

// broadcast 向所有WebSocket客户端广播消息
func (ws *WSLogAPI) broadcast(message string) {
	ws.clientsMux.Lock()
	defer ws.clientsMux.Unlock()

	for conn := range ws.clients {
		err := conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			conn.Close()
			delete(ws.clients, conn)
		}
	}
}
