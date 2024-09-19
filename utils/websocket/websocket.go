package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type Connections struct {
	Clients          map[*websocket.Conn]bool
	IPAddresses      map[string]*websocket.Conn
	Broadcast        chan []byte
	ConnectionUpdate chan int
	mu               sync.RWMutex
}

type ConnectionMessage struct {
	Connections int `json:"connections"`
}

var connections Connections

func init() {
	connections = Connections{
		Clients:          make(map[*websocket.Conn]bool),
		IPAddresses:      make(map[string]*websocket.Conn),
		Broadcast:        make(chan []byte),
		ConnectionUpdate: make(chan int),
	}
	go handleMessages()
	go handleConnectionUpdates()
}

func WebsocketHandler(c echo.Context) error {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Println("Error while upgrading connection:", err)
		return err
	}
	defer func() {
		conn.Close()
		removeConnection(conn, c.RealIP())
	}()

	if !NewConnection(c.RealIP(), conn) {
		conn.Close()
		return echo.NewHTTPError(http.StatusForbidden, "Only one connection per IP address is allowed")
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error while reading message:", err)
			break
		}

		if err := PushMessage(conn, msg); err != nil {
			log.Println("Error while writing message:", err)
			break
		}
	}
	return nil
}

func NewConnection(ip string, conn *websocket.Conn) bool {
	connections.mu.Lock()
	defer connections.mu.Unlock()

	if _, exists := connections.IPAddresses[ip]; exists {
		return false
	}

	connections.Clients[conn] = true
	connections.IPAddresses[ip] = conn
	connections.ConnectionUpdate <- len(connections.Clients)
	PushMessage(conn, []byte("connected to achan.moe"))
	return true
}

func removeConnection(conn *websocket.Conn, ip string) {
	connections.mu.Lock()
	defer connections.mu.Unlock()
	delete(connections.Clients, conn)
	delete(connections.IPAddresses, ip)
	connections.ConnectionUpdate <- len(connections.Clients)
}

func handleMessages() {
	for {
		msg := <-connections.Broadcast
		connections.mu.RLock()
		for conn := range connections.Clients {
			if err := PushMessage(conn, msg); err != nil {
				log.Println("Error while writing message:", err)
				removeConnection(conn, conn.RemoteAddr().String())
			}
		}
		connections.mu.RUnlock()
	}
}

func handleConnectionUpdates() {
	for {
		numConnections := <-connections.ConnectionUpdate
		message := ConnectionMessage{Connections: numConnections}
		msg, err := json.Marshal(message)
		if err != nil {
			log.Println("Error while marshaling JSON:", err)
			continue
		}

		connections.mu.RLock()
		for conn := range connections.Clients {
			if err := PushMessage(conn, msg); err != nil {
				log.Println("Error while writing message:", err)
				removeConnection(conn, conn.RemoteAddr().String())
			}
		}
		connections.mu.RUnlock()
	}
}

func PushMessage(conn *websocket.Conn, msg []byte) error {
	return conn.WriteMessage(websocket.TextMessage, msg)
}
