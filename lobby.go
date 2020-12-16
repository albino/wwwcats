package main

import (
	"log"
)

type Lobby struct {
	name string

	// Client management
	clients map[*Client]bool
	register chan *Client
	unregister chan *Client

	// Chat
	chat chan string
}

func newLobby(name string) *Lobby {
	return &Lobby{
		name: name,
		clients: make(map[*Client]bool),
		register: make(chan *Client),
		unregister: make(chan *Client),
		chat: make(chan string),
	}
}

func (l *Lobby) run(lobbies map[string]*Lobby) {
	// Goroutine to deal with all the tasks of the lobby

	for {
		select {
		case client := <-l.register:
			l.clients[client] = true
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
		// TODO: chat
		}
	}
}

func (l *Lobby) readFromClient(message string) {
	log.Println(message)
}
