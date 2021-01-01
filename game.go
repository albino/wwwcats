package main

import (
	"strings"
	"strconv"
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

	// Various game-related variables
	defusing bool
	attack bool

	// It's easier if we index the spectators, so we use a map
	// Only synced with the client during netburst
	spectators map[*Client]bool

	deck *Deck
	hands map[*Client]*Hand

	// We store one action's worth of 'history' in case of a nope
	history *GameState
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
	g.lobby.sendBcast("joins "+client.name)
	g.netburst(client)
}

func (g *Game) removePlayer(client *Client) {
	_, ok := g.spectators[client]
	if !ok {
		// First downgrade the player to spectator
		g.downgradePlayer(client)
	}

	delete(g.spectators, client)
	g.lobby.sendBcast("parts "+client.name)
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
	currentlyPlaying := false
	if g.playerNumber(client) == g.currentPlayer {
		currentlyPlaying = true
	}

	clientToRemove := g.playerNumber(client)
	g.players = append(g.players[:clientToRemove], g.players[clientToRemove+1:]...)
	g.spectators[client] = true
	delete(g.hands, client)

	g.lobby.sendBcast("downgrades "+client.name)
	g.lobby.sendBcast("players" + g.playerList())

	if (!g.started) {
		// Display a message to tell the client they are spectating
		client.sendMsg("message spectating");
		return
	}

	// Gracefully remove the player from the game in progress
	client.sendMsg("message spectating_exploded");

	// Erase their hand
	client.sendMsg("hand")

	if len(g.players) == 1 {
		go g.wins(g.players[0])
		return
	}

	// If they are currently playing, advance to the next player
	if currentlyPlaying && len(g.players) > 0 {
		g.nextTurn()
	}
}

func (g *Game) wins(winner *Client) {
	g.lobby.sendBcast("wins "+winner.name)

	// This function runs a separate goroutine, so it's safe to sleep
	time.Sleep(5 * time.Second)

	// Destroy the game and create a new one
	g.lobby.sendBcast("hand")
	g.lobby.sendBcast("draw_pile no")
	g.lobby.sendBcast("no_discard")
	g.lobby.sendBcast("bcast new_game")

	g.lobby.currentGame = newGame(g.lobby)
	for client := range g.lobby.clients {
		g.lobby.currentGame.addPlayer(client)
		// We add a very short delay to allow each joining client to be processed separately
		time.Sleep(50 * time.Millisecond)
	}

	// The GC should now be able to collect this old game object, I think
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

		_, ok := g.spectators[c]
		if ok {
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
		_, ok := g.spectators[c]
		if ok {
			return
		}

		if g.currentPlayer >= len(g.players) {
			return
		}

		if g.deck.cardsLeft() < 1 {
			c.sendMsg("err illegal_move")
			return
		}

		if g.players[g.currentPlayer].name != c.name {
			c.sendMsg("err illegal_move")
			return
		}

		if g.defusing {
			return
		}

		g.drawCard(c)
		return
	}

	if fields[0] == "play" {
		_, ok := g.spectators[c]
		if ok {
			return
		}

		if g.currentPlayer >= len(g.players) {
			return
		}

		card, err := strconv.Atoi(fields[1])
		if err != nil {
			c.sendMsg("err illegal_move")
			return
		}

		if card >= g.hands[c].getLength() {
			c.sendMsg("err illegal_move")
			return
		}

		cardText := g.hands[c].getCard(card)

		if g.defusing && cardText != "defuse" {
			return
		}

		if cardText != "nope" && g.players[g.currentPlayer].name != c.name {
			c.sendMsg("err illegal_move")
			return
		}

		g.hands[c].removeCard(card)
		c.sendMsg("hand" + g.hands[c].cardList())
		g.playsCard(c, cardText)

		return
	}

	if fields[0] == "a" {
		g.answersQuestion(c, fields[1], fields[2])
		return
	}

	log.Println("Uncaught message from", c.name + ":", msg)
}

func (g *Game) drawCard(c *Client) {
	card := g.deck.draw()
	g.history = nil

	if card == "exploding" {
		g.lobby.sendBcast("exploded "+c.name)

		if !g.hands[c].contains("defuse") {
			g.downgradePlayer(c)
			return
		}

		g.defusing = true
		c.sendMsg("defusing")

		g.nextTurn()
		return
	}

	g.hands[c].addCard(card)
	c.sendMsg("hand" + g.hands[c].cardList())
	// Tell the player what card they drew
	c.sendMsg("drew "+card)
	// Tell everyone else that a mystery card was drawn
	g.lobby.sendComplexBcast("drew_other "+c.name, map[*Client]bool{c: true})

	g.incrementTurn()
	g.nextTurn()
}

func (g *Game) playsCard(player *Client, card string) {
	g.lobby.sendBcast("played "+player.name+" "+card)

	switch card {
	case "defuse":
		if !g.defusing {
			return
		}

		player.sendMsg("q defuse_pos")
	case "shuffle":
		g.history = makeGameState(g)
		g.deck.shuffle()
	case "nope":
		if g.history == nil {
			g.lobby.sendBcast("bcast no_nope")
			return
		}

		g.history.restore(g)
		g.nextTurn()
	case "skip":
		g.history = makeGameState(g)
		g.incrementTurn()
		g.nextTurn()
	case "attack":
		g.history = makeGameState(g)
		if g.attack {
			// player is on the first turn of an attack
			g.attack = false
		} else {
			g.currentPlayer++
			g.attack = true
		}
		g.nextTurn()
	default:
		log.Println("unhandled card: ", card)
	}
}

func (g *Game) answersQuestion(player *Client, question string, answer string) {
	switch question {
	case "defuse_pos":
		if !g.defusing {
			return
		}

		if g.players[g.currentPlayer] != player {
			return
		}

		pos, err := strconv.Atoi(answer)
		if err != nil {
			player.sendMsg("q "+question)
			return
		}

		if pos > g.deck.cardsLeft() {
			player.sendMsg("q "+question)
			return
		}

		g.deck.insertAtPos(pos, "exploding")
		g.defusing = false
		g.incrementTurn()
		g.nextTurn()
	default:
		log.Println("unexpected Q/A: ", question, answer)
	}
}

func (g *Game) incrementTurn() {
	// Changes the turn counter to the next player
	// (or not, if an attack has been played)

	if g.attack {
		g.attack = false
		return
	}

	g.currentPlayer++

	return
}

func (g *Game) nextTurn() {
	// Begins the next turn
	// NB. this doesn't change currentPlayer

	if g.currentPlayer >= len(g.players) {
		g.currentPlayer = 0
	}

	g.lobby.sendBcast("now_playing "+g.players[g.currentPlayer].name)
	if g.deck.cardsLeft() == 0 {
		g.lobby.sendBcast("draw_pile no")
	} else {
		g.lobby.sendBcast("draw_pile yes")
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
