package main

import (
	"time"
	"log"

	"github.com/gorilla/websocket"
)

const (
	// Timeout for writing to a client
	writeWait = 10 * time.Second

	// Timeout for receiving a 'pong' from the client
	pongWait = 15 * time.Second

	// How often should we ping? Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 512
)

type Client struct {
	// Websocket connection object
	conn *websocket.Conn

	// Buffer for outgoing messages
	send chan []byte
}

func (c *Client) readPump() {
	// Sets up a client, reads incoming messages and sends them to the right place
	//
	// This is called as a goroutine for each client, and this function
	// is the only function allowed to read from the client.

	defer func() {
		// Clean up
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			// The connection is dead
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("err: %v", err)
			}
			log.Println("goodbye!")
			break
		}
		select {
		case c.send <- message:
		default:
			close(c.send)
		}
	}
}

func (c *Client) writePump() {
	// Counterpart to readPump
	// Massively adapted from the gorilla websocket docs

	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				// Close the channel
				// I have no idea how this actually works
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Finish writing whatever is left
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
