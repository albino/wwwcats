package main

import (
	"log"
	"strconv"
	"strings"
	"time"

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

	name  string
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

		// Read the incoming messages

		_, bytes, err := c.conn.ReadMessage()
		if err != nil {
			// The connection is dead
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("%s !!! Connection closed (%v)", c.name, err)
			}
			break
		}

		message := string(bytes)

		// Check for badly-formed messages which could do something strange
		if strings.Contains(message, "\n") || strings.Contains(message, "\r") {
			continue
		}

		log.Printf("%s >>> %s", c.name, message)

		// If this client is in a lobby, let the lobby handle the message

		if c.lobby != nil {
			c.lobby.readFromClient(c, message)
			continue
		}

		// The client is not currently in a lobby; check if they're trying to join

		fields := strings.Fields(message)

		if len(fields) == 3 && fields[0] == "join_lobby" {
			lobby_name := fields[1]
			player_name := fields[2]

			// Length is already limited by SetReadLimit, so we're not worried

			c.joinToLobby(lobby_name, player_name, lobbies)
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

	// Send the server version on connect
	c.sendMsg("version "+strconv.Itoa(REVISION))

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				log.Printf("%s !!! Write channel closed", c.name)

				// Close the channel
				// I have no idea how this actually works
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("%s !!! Disconnected on write (%v)", c.name, err)
				return
			}
			w.Write(message)

			/*
				// Finish writing whatever is left
				n := len(c.send)
				for i := 0; i < n; i++ {
					w.Write([]byte("\r\n"))
					w.Write(<-c.send)
				}
			*/

			if err := w.Close(); err != nil {
				log.Printf("%s !!! Couldn't close write (%v)", c.name, err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("%s !!! Ping timeout (%v)", c.name, err)
				return
			}
		}
	}
}

func (c *Client) sendMsg(message string) {
	log.Printf("%s <<< %s", c.name, message)

	select {
	case c.send <- []byte(message):
	default:
		log.Fatal("Failed to write to client", c.name)
	}
}
