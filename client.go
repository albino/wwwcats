package main

import (
	"time"
	"log"
	"strings"

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

	lobby *Lobby
}

func (c *Client) readPump(lobbies map[string]*Lobby) {
	// Sets up a client, reads incoming messages and sends them to the right place
	//
	// This is called as a goroutine for each client, and this function
	// is the only function allowed to read from the client.

	defer func() {
		// Clean up
		c.conn.Close()
		if c.lobby != nil {
			c.lobby.unregister <- c
		}
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, bytes, err := c.conn.ReadMessage()
		if err != nil {
			// The connection is dead
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("err: %v", err)
			}
			break
		}

		message := string(bytes)

		if c.lobby != nil {
			c.lobby.readFromClient(message)
			continue
		}

		// The client is not currently in a lobby; check if they're trying to join

		fields := strings.Fields(message)

		if fields[0] == "join_lobby" && len(fields) == 2 {
			lobby_name := fields[1]

			// Length is already limited by SetReadLimit, so we're not worried

			var lobby *Lobby

			if lobbies[lobby_name] == nil {
				// Create the lobby, and start its goroutine
				lobby = newLobby(lobby_name)
				lobbies[lobby_name] = lobby
				go lobby.run(lobbies)
			} else {
				lobby = lobbies[lobby_name]
			}

			lobby.register <- c
			c.lobby = lobby
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
