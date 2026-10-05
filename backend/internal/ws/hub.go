package ws

import (
	"time"

	"github.com/gorilla/websocket"
)

// 定义客户端 struct
type Client struct {
	send   chan []byte
	hub    *Hub
	userID uint64
	conn   *websocket.Conn
}

type broadcastMsg struct {
	userID uint64
	data   []byte
}

// 定义hub 结构体
type Hub struct {
	clients    map[uint64]map[*Client]bool // 在线的链接池
	Register   chan *Client                // 注册：有人要加入
	unregister chan *Client                // 注销：有人要离开
	broadcast  chan broadcastMsg           // 广播：要发给所有人的消息
}

// 实例化hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint64]map[*Client]bool),
		Register:   make(chan *Client, 32),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan broadcastMsg, 256),
	}
}

// 实例化客户端
func NewClient(hub *Hub, userID uint64, conn *websocket.Conn) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		userID: userID,
		send:   make(chan []byte, 256),
	}
}

// hub 运行方法
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			// 有人加入： 加到 map 里
			if _, ok := h.clients[client.userID]; !ok {
				h.clients[client.userID] = make(map[*Client]bool)
			}
			h.clients[client.userID][client] = true
		case client := <-h.unregister:

			conns, ok := h.clients[client.userID]
			if ok {
				if _, exists := conns[client]; exists {
					delete(conns, client)
					close(client.send)
					if len(conns) == 0 {
						delete(h.clients, client.userID)
					}
				}
			}
			// 有人离开： 从 map 里删掉，关闭 client 的 send channel
		case msg := <-h.broadcast:
			// 广播： 遍历 map，给每个 client 的 send channel 发消息
			coons, ok := h.clients[msg.userID]
			if !ok {
				continue
			}

			for client := range coons {
				select {
				case client.send <- msg.data:
				default:
					// 写不进去（send 满了）踢掉这个 client
					delete(coons, client)
					close(client.send)
				}
			}
		}
	}
}

// 超时时间
const pongWait = 60 * time.Second

// 持续从websocket 连接读消息
func (c *Client) ReadPump() {
	defer c.CloseClient()

	// 设置第一次读取的超时时间
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	// 收到客户端 Pong 后，刷新超时时间
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		// c.conn 读消息
		_, message, err := c.conn.ReadMessage()

		if err != nil {
			select {
			case c.hub.unregister <- c:
			default:
			}
			return
		}

		select {
		case c.hub.broadcast <- broadcastMsg{userID: c.userID, data: message}:
		default:
		}

	}
}

const (
	writeWait  = 5 * time.Second
	pingPeriod = 54 * time.Second
)

// 监听 send channel,有消息就写给客户端
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.CloseClient()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				return
			}
		}
	}
}

// 公开通知方法
func (h *Hub) BroadcastToUser(userID uint64, data []byte) {
	h.broadcast <- broadcastMsg{userID: userID, data: data}
}

// 统一关闭方法
func (c *Client) CloseClient() {
	c.conn.Close()
}
