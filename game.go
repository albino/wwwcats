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
	currentPlayer int

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
		currentPlayer: -1,
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
	if !g.started {
		client.sendMsg("message spectating");
		return
	}

	// allow the client to spectate a game-in-progress
	client.sendMsg("message spectating_started")
	if g.deck.cardsLeft() > 0 {
		client.sendMsg("draw_pile yes")
	}
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

	if fields[0] == "draw" {
		if g.deck.cardsLeft() < 1 {
			c.sendMsg("err illegal_move")
			return
		}

		if g.players[g.currentPlayer].name != c.name {
			c.sendMsg("err illegal_move")
			return
		}

		g.drawCard(c)
		return
	}

	log.Println("Uncaught message from", c.name + ":", msg)
}

func (g *Game) drawCard(c *Client) {
	card := g.deck.draw()

	if card == "exploding" {
		// TODO what if a player explodes?
	}

	g.hands[c].addCard(card)
	c.sendMsg("hand" + g.hands[c].cardList())
	// Tell the player what card they drew
	c.sendMsg("drew "+card)
	// Tell everyone else that a mystery card was drawn
	g.lobby.sendComplexBcast("drew_other "+c.name, map[*Client]bool{c: true})

	// End this player's turn
	g.currentPlayer++
	if g.currentPlayer >= len(g.players) {
		g.currentPlayer = 0
	}
	g.lobby.sendBcast("now_playing "+g.players[g.currentPlayer].name)
	if g.deck.cardsLeft() == 0 {
		g.lobby.sendBcast("draw_pile no")
	}
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
	g.currentPlayer = 0
	g.lobby.sendBcast("players" + g.playerList())
	g.lobby.sendBcast("now_playing "+g.players[g.currentPlayer].name)

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
