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
	Broadcast        chan []byte
	ConnectionUpdate chan int
	mu               sync.Mutex
}

type ConnectionMessage struct {
	Connections int    `json:"connections"`
	Time        string `json:"time,omitempty"`
	Uptime      string `json:"uptime,omitempty"`
}

var connections Connections

func init() {
	connections = Connections{
		Clients:          make(map[*websocket.Conn]bool),
		Broadcast:        make(chan []byte),
		ConnectionUpdate: make(chan int),
	}
	go handleMessages()
	go handleConnectionUpdates()
}

func GetConnection(c echo.Context) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Println("Error while upgrading connection:", err)
		return nil, err
	}
	return conn, nil
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
		delete(connections.Clients, conn)
		connections.ConnectionUpdate <- len(connections.Clients)
	}()

	NewConnection(conn)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error while reading message:", err)
			break
		}

		err = PushMessage(conn, msg)
		if err != nil {
			log.Println("Error while writing message:", err)
			break
		}
	}
	return nil
}

func NewConnection(conn *websocket.Conn) {
	connections.Clients[conn] = true
	connections.ConnectionUpdate <- len(connections.Clients)
	PushMessage(conn, []byte("Welcome to the WebSocket server!"))
}

func handleMessages() {
	for {
		msg := <-connections.Broadcast
		for conn := range connections.Clients {
			err := PushMessage(conn, msg)
			if err != nil {
				log.Println("Error while writing message:", err)
				conn.Close()
				delete(connections.Clients, conn)
				connections.ConnectionUpdate <- len(connections.Clients)
			}
		}
	}
}

func handleConnectionUpdates() {
	for {
		numConnections := <-connections.ConnectionUpdate
		message := ConnectionMessage{
			Connections: numConnections,
		}
		msg, err := json.Marshal(message)
		if err != nil {
			log.Println("Error while marshaling JSON:", err)
			continue
		}
		for conn := range connections.Clients {
			err := PushMessage(conn, msg)
			if err != nil {
				log.Println("Error while writing message:", err)
				conn.Close()
				delete(connections.Clients, conn)
				connections.ConnectionUpdate <- len(connections.Clients)
			}
		}
	}
}

func PushMessage(conn *websocket.Conn, msg []byte) error {
	connections.mu.Lock()
	defer connections.mu.Unlock()
	err := conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		log.Println("Error while pushing message:", err)
		return err
	}
	return nil
}
