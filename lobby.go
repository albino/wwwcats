package main

import (
	"log"
	"strings"
)

type Lobby struct {
	name string

	// Client management
	clients map[*Client]bool
	register chan *Client
	unregister chan *Client

	// Broadcast
	bcast chan []byte

	currentGame *Game
}

func newLobby(name string) (lobby *Lobby) {
	lobby = &Lobby{
		name: name,
		clients: make(map[*Client]bool),

		// We make channels with a small buffer, in case we need to
		// write to them from their own goroutine for convenience

		register: make(chan *Client, 2),
		unregister: make(chan *Client, 2),
		bcast: make(chan []byte, 2),
	}
	lobby.currentGame = newGame(lobby)
	return
}

func (l *Lobby) run(lobbies map[string]*Lobby) {
	// Goroutine to deal with all the tasks of the lobby

	for {
		select {

		case client := <-l.register:
			l.clients[client] = true

			// Announce the join
			l.sendBcast("joins "+client.name)

			// Sync the join to the game object
			l.currentGame.addPlayer(client)

			// Sync the game state to the new client
			l.currentGame.netburst(client)

		case client := <-l.unregister:
			if _, ok := l.clients[client]; ok {
				delete(l.clients, client)
				close(client.send)
			}

			if len(l.clients) == 0 {
				// The lobby is finished
				delete(lobbies, l.name)
				return
			}

			// Announce and sync
			l.sendBcast("parts "+client.name)
			l.currentGame.removePlayer(client)

		case bytes := <-l.bcast:
			for client := range l.clients {
				select {
				case client.send <- bytes:
				default:
					l.unregister <- client
				}
			}

		}
	}
}

func (c *Client) joinToLobby(lobby_name string, player_name string, lobbies map[string]*Lobby) {
	var lobby *Lobby

	if lobbies[lobby_name] == nil {
		// Create the lobby, and start its goroutine
		lobby = newLobby(lobby_name)
		lobbies[lobby_name] = lobby
		go lobby.run(lobbies)
	} else {
		lobby = lobbies[lobby_name]
	}

	// Avoid nickname collisions
	for connected_client := range lobbies[lobby_name].clients {
		if connected_client.name == player_name {
			select {
			case c.send <- []byte("err username_exists"):
			default:
				close(c.send)
			}
			// Nobody is getting joined to the lobby today
			return
		}
	}

	c.name = player_name

	lobby.register <- c
	c.lobby = lobby
}

func (l *Lobby) readFromClient(c *Client, msg string) {
	fields := strings.Fields(msg)

	// Lobby-wide commands

	if fields[0] == "chat" {
		l.sendBcast("chat "+c.name+" "+msg[5:])
		return
	}

	// Nothing to be done here, hand the message off to the game object
	l.currentGame.readFromClient(c, msg)
}

func (l *Lobby) sendBcast(msg string) {
	select {
	case l.bcast <- []byte(msg):
	default:
		log.Fatal("failed to broadcast message", msg)
	}
}
