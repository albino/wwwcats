package main

import (
	"strings"
	"log"
	"math/rand"
	"time"
)

type Game struct {
	lobby *Lobby

	started bool

	// We need a strict order for players, so we use a slice
	// The player list is order-sensitive, so we re-sync it
	// with the client every time it's updated.
	players []*Client

	// It's easier if we index the spectators, so we use a map
	// Only synced with the client during netburst
	spectators map[*Client]bool

	deck *Deck
	hands map[*Client]*Hand
}

func newGame(lobby *Lobby) *Game {
	return &Game {
		lobby: lobby,
		spectators: make(map[*Client]bool),
		hands: make(map[*Client]*Hand),
	}
}

func (g *Game) addPlayer(client *Client) {
	g.spectators[client] = true
}

func (g *Game) removePlayer(client *Client) {
	_, ok := g.spectators[client]
	if ok {
		// Remove the player from the spectators
		delete(g.spectators, client)
	} else {
		// Remove the player from the players
		clientToRemove := g.playerNumber(client)
		g.players = append(g.players[:clientToRemove], g.players[clientToRemove+1:]...)

		// synchronise a new player list with all the clients
		g.lobby.sendBcast("players" + g.playerList())

		// TODO what if the game has started?
	}
}

func (g *Game) playerNumber(client *Client) (num int) {
	// Finds the number of a player in the players slice
	// Mostly useful for removing players
	num = -1
	for i, _ := range g.players {
		if g.players[i] == client {
			num = i
			break
		}
	}
	return
}

func (g *Game) upgradePlayer(client *Client) {
	// Move a player from the spectators into the players
	delete(g.spectators, client)
	g.players = append(g.players, client)

	g.lobby.sendBcast("upgrades "+client.name)
	g.lobby.sendBcast("players" + g.playerList())

	// Display a message to tell the client they are playing
	client.sendMsg("message playing");
}

func (g *Game) downgradePlayer(client *Client) {
	clientToRemove := g.playerNumber(client)
	g.players = append(g.players[:clientToRemove], g.players[clientToRemove+1:]...)
	g.spectators[client] = true

	g.lobby.sendBcast("downgrades "+client.name)
	g.lobby.sendBcast("players" + g.playerList())

	// Display a message to tell the client they are spectating
	client.sendMsg("message spectating");

	// TODO what if the game has started?
}

func (g *Game) netburst(client *Client) {
	// Communicates the current game state to a newly joining client
	client.sendMsg("spectators" + g.spectatorList())
	client.sendMsg("players" + g.playerList())

	// Display a message to tell the client they are spectating
	// TODO: a different message if the game has already started
	client.sendMsg("message spectating");
}

func (g *Game) spectatorList() (list string) {
	for spec := range g.spectators {
		list = list + " " + spec.name
	}
	return
}

func (g *Game) playerList() (list string) {
	for _, player := range g.players {
		list = list + " " + player.name
	}
	return
}

func (g *Game) readFromClient(c *Client, msg string) {
	fields := strings.Fields(msg)

	if fields[0] == "join" {
		// Joining the game (from spectators)

		if g.started {
			// You can't join an active game
			return
		}

		_, ok := g.spectators[c]
		if !ok {
			// Player is not a spectator!
			return
		}

		g.upgradePlayer(c)
		return
	}

	if fields[0] == "leave" {
		// Leaving the game (back to spectators)

		n := g.playerNumber(c)
		if n == -1 {
			// Player is a spectator!
			return
		}

		g.downgradePlayer(c)
		return
	}

	if fields[0] == "start" {
		if g.started {
			return
		}

		if len(g.players) < 2 {
			return
		}

		g.start()
		return
	}

	log.Println("Uncaught message from", c.name + ":", msg)
}

func (g *Game) start() {
	// Starts the game

	// TODO: maximum no of players

	g.started = true

	g.lobby.sendBcast("clear_message");
	g.lobby.sendBcast("bcast starting");

	// Shuffle the player list and re-send it
	rand.Seed(time.Now().UnixNano()) // Hard enough to predict; should be fine
	rand.Shuffle(len(g.players), func (i, j int) {
		g.players[i], g.players[j] = g.players[j], g.players[i]
	})
	g.lobby.sendBcast("players" + g.playerList())

	// Generate the deck
	g.deck = newDeck()

	// Give each player a hand
	for _, player := range g.players {
		g.hands[player] = g.deck.dealHand()
	}

	// Shuffle in extra cards
	g.deck.addExtraCards(len(g.players))

	// Sync the cards to the client
	for _, player := range g.players {
		player.sendMsg("hand" + g.hands[player].cardList())
	}
	g.lobby.sendBcast("draw_pile yes")
}
