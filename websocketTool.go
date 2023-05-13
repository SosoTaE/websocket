package main

import (
	"log"
	"strconv"
	"sync/atomic"
	"github.com/gorilla/websocket"

)


var connectionCounter uint64

func generateConnectionID() string {
	id := atomic.AddUint64(&connectionCounter, 1)
	return "conn" + strconv.FormatUint(id, 10)
}


type client struct {
	conn websocket.Conn
	send chan []byte
}


var clients = make(map[*client]bool)
var broadcast = make(chan []byte)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}

	client := &client{
		conn: conn,
		send: make(chan []byte),
	}
	clients[client] = true

	go client.writePump()
	go client.readPump()
}

func (c *client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return
			}
			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Println("Error writing message:", err)
				return
			}
		}
	}
}


func (c *client) readPump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Println("Error reading message:", err)
			}
			break
		}

		broadcast <- message
	}
}


func broadcastMessages() {
	for {
		message := <-broadcast
		for client := range clients {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(clients, client)
			}
		}
	}
}
