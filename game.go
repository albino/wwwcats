package main

import (
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type Game struct {
	// The corresponding Lobby object, to allow communication
	// with clients
	lobby *Lobby

	// It's easier if we index the spectators, so we use a map
	// Only synced with the client during netburst
	spectators map[*Client]bool

	// We need a strict order for players, so we use a slice
	// The player list is order-sensitive, so we re-sync it
	// with the client every time it's updated.
	players       []*Client
	currentPlayer int

	// Various game-related variables
	started   bool
	defusing  bool
	attack    bool
	favouring *Client // Who is asking for a favour?
	favoured  *Client // Who is being asked for a favour?
	favourType int // !! not reset !!
	// 1 - favour, 2 - random, 3 - steal

	deck  *Deck
	hands map[*Client]*Hand

	// We store one action's worth of 'history' in case of a nope
	history *GameState
}

func newGame(lobby *Lobby) *Game {
	return &Game{
		lobby:         lobby,
		spectators:    make(map[*Client]bool),
		hands:         make(map[*Client]*Hand),
		currentPlayer: -1,
	}
}

func (g *Game) addPlayer(client *Client) {
	g.spectators[client] = true
	g.lobby.sendBcast("joins " + client.name)
	g.netburst(client)
}

func (g *Game) removePlayer(client *Client) {
	_, ok := g.spectators[client]
	if !ok {
		// First downgrade the player to spectator
		g.downgradePlayer(client)
	}

	delete(g.spectators, client)
	g.lobby.sendBcast("parts " + client.name)
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

func (g *Game) playerByName(name string) (player *Client) {
	for _, thisPlayer := range g.players {
		if thisPlayer.name == name {
			player = thisPlayer
			break
		}
	}
	return
}

func (g *Game) upgradePlayer(client *Client) {
	// Move a player from the spectators into the players
	delete(g.spectators, client)
	g.players = append(g.players, client)

	g.lobby.sendBcast("upgrades " + client.name)
	g.lobby.sendBcast("players" + g.playerList())

	// Display a message to tell the client they are playing
	client.sendMsg("message playing")
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

	g.lobby.sendBcast("downgrades " + client.name)
	g.lobby.sendBcast("players" + g.playerList())

	if !g.started {
		// Display a message to tell the client they are spectating
		client.sendMsg("message spectating")
		return
	}

	// Gracefully remove the player from the game in progress
	client.sendMsg("message spectating_exploded")

	// Erase their hand
	client.sendMsg("hand")

	if len(g.players) == 1 {
		go g.wins(g.players[0])
		return
	}

	if g.favouring == client {
		g.favoured.sendMsg("q_cancel")
		g.favouring = nil
		g.favoured = nil
	}
	if g.favoured == client && g.favourType == 1 {
		// The favour is cancelled
		g.lobby.sendBcast("bcast favour_cancel")
		g.favouring.sendMsg("unlock")
		g.favouring = nil
		g.favoured = nil
	}
	// for favourType 2, it is instant - this can't happen
	// for favourType 3, we are waiting on a response from the player ASKING
	// (so we can deal with it in answersQuestion)

	// If they are currently playing, advance to the next player
	if currentlyPlaying && len(g.players) > 0 {
		g.nextTurn()
	}
}

func (g *Game) wins(winner *Client) {
	g.lobby.sendBcast("wins " + winner.name)

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
		client.sendMsg("message spectating")
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

	switch fields[0] {
	case "join":
		// Joining the game (from spectators)

		if g.started {
			// You can't join an active game
			break
		}

		_, ok := g.spectators[c]
		if !ok {
			// Player is not a spectator!
			break
		}

		g.upgradePlayer(c)

	case "leave":
		// Leaving the game (back to spectators)

		_, ok := g.spectators[c]
		if ok {
			// Player is a spectator!
			break
		}

		g.downgradePlayer(c)

	case "start":
		if g.started {
			break
		}

		if len(g.players) < 2 {
			c.sendMsg("bcast min_players")
			break
		}

		if len(g.players) > 6 {
			g.lobby.sendBcast("bcast max_players")
			break
		}

		if len(g.players) == 6 {
			// Warning message
			g.lobby.sendBcast("bcast high_players")
		}

		g.start()

	case "draw":
		_, ok := g.spectators[c]
		if ok {
			break
		}

		if g.currentPlayer >= len(g.players) {
			break
		}

		if g.deck.cardsLeft() < 1 {
			c.sendMsg("err illegal_move")
			break
		}

		if g.players[g.currentPlayer].name != c.name {
			c.sendMsg("err illegal_move")
			break
		}

		if g.defusing {
			break
		}

		g.drawCard(c)

	case "play":
		_, ok := g.spectators[c]
		if ok {
			break
		}

		if g.currentPlayer >= len(g.players) {
			break
		}

		if g.favouring != nil {
			c.sendMsg("err illegal_move")
			break
		}

		card, err := strconv.Atoi(fields[1])
		if err != nil {
			c.sendMsg("err illegal_move")
			break
		}

		if card >= g.hands[c].getLength() {
			c.sendMsg("err illegal_move")
			break
		}

		cardText := g.hands[c].getCard(card)

		if g.defusing && cardText != "defuse" {
			break
		}

		if cardText != "nope" && g.players[g.currentPlayer].name != c.name {
			c.sendMsg("err illegal_move")
			break
		}

		g.favouring = nil
		g.favoured = nil
		g.hands[c].removeCard(card)
		c.sendMsg("hand" + g.hands[c].cardList())
		g.playsCard(c, cardText)

	case "play_multiple":
		_, ok := g.spectators[c]
		if ok {
			break
		}

		if g.currentPlayer >= len(g.players) {
			break
		}

		if g.favouring != nil {
			c.sendMsg("err illegal_move")
			break
		}

		num, err := strconv.Atoi(fields[1])
		if err != nil || num > 3 || num < 2 {
			c.sendMsg("err illegal_move")
			break
		}

		if !g.hands[c].containsMultiple(fields[2], num) {
			c.sendMsg("err illegal_move")
			break
		}

		g.favouring = nil
		g.favoured = nil
		for i := 0; i < num; i++ {
			g.hands[c].removeByName(fields[2])
		}
		c.sendMsg("hand" + g.hands[c].cardList())
		g.playsCombo(c, fields[2], num)

	case "a":
		g.answersQuestion(c, fields[1], fields[2])

	case "sort":
		_, ok := g.spectators[c]
		if ok {
			break
		}
		g.hands[c].sort()
		c.sendMsg("hand" + g.hands[c].cardList())

	default:
		log.Println("Uncaught message from", c.name+":", msg)
	} // End switch
}

func (g *Game) drawCard(c *Client) {
	card := g.deck.draw()
	g.history = nil
	g.favouring = nil
	g.favoured = nil

	if card == "exploding" {
		g.lobby.sendBcast("exploded " + c.name)

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
	c.sendMsg("drew " + card)
	// Tell everyone else that a mystery card was drawn
	g.lobby.sendComplexBcast("drew_other "+c.name, map[*Client]bool{c: true})

	g.incrementTurn()
	g.nextTurn()
}

func (g *Game) playsCard(player *Client, card string) {
	g.lobby.sendBcast("played " + player.name + " " + card)

	// Every case here should do something with g.history
	// - either make a backup, or clear it so that we can't
	// NOPE too far back into history
	switch card {
	case "defuse":
		if !g.defusing {
			return
		}
		player.sendMsg("q defuse_pos")
	case "favour":
		g.favouring = player
		g.favourType = 1
		player.sendMsg("q favour_who")
	case "shuffle":
		g.history = makeGameState(g)
		g.deck.shuffle()
	case "nope":
		if g.history == nil {
			g.lobby.sendBcast("bcast no_nope")
			return
		}
		// Back up the game state before restoring the old one
		// - that way, you can NOPE a NOPE
		history := makeGameState(g)
		g.history.restore(g)
		g.history = history
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
	case "see3":
		cards := g.deck.peek(3)
		player.sendMsg("seen " + strings.Join(cards, " "))
		g.history = nil
	default:
		log.Println("unhandled card: ", card)
	}
}

func (g *Game) playsCombo(player *Client, card string, num int) {
	g.lobby.sendBcast("played_multiple " + player.name + " " + strconv.Itoa(num) + " " + card)

	g.history = nil // TODO

	if num == 2 {
		// 2 of a kind - random card
		g.favouring = player
		g.favourType = 2
		player.sendMsg("q random_who")
	} else if num == 3 {
		// 3 of a kind - 'stealing' a card
		g.favouring = player
		g.favourType = 3
		player.sendMsg("q steal_who")
	} else {
		log.Fatal("what the chuff??")
	}
}

func (g *Game) answersQuestion(player *Client, question string, answer string) {
	switch question {
	case "defuse_pos":
		if !g.defusing {
			break
		}

		if g.players[g.currentPlayer] != player {
			break
		}

		pos, err := strconv.Atoi(answer)
		if err != nil {
			player.sendMsg("q " + question)
			break
		}

		if pos > g.deck.cardsLeft() {
			player.sendMsg("q " + question)
			break
		}

		g.deck.insertAtPos(pos, "exploding")
		g.defusing = false
		g.incrementTurn()
		g.nextTurn()
	case "favour_who":
		if g.favouring != player {
			break
		}

		target := g.playerByName(answer)
		if target == nil || target == player {
			player.sendMsg("q " + question)
			break
		}

		g.favoured = target
		g.lobby.sendComplexBcast("favoured "+player.name+" "+target.name, map[*Client]bool{target: true})
		target.sendMsg("q favour_what " + player.name)
		player.sendMsg("lock") // block further play until the transaction completes
	case "random_who":
		if g.favouring != player {
			break
		}

		target := g.playerByName(answer)
		if target == nil || target == player {
			player.sendMsg("q " + question)
			break
		}

		// removes the card from the target's hand
		card := g.hands[target].takeRandom()

		g.lobby.sendComplexBcast("randomed "+player.name+" "+target.name,
			map[*Client]bool{target: true, player: true})
		target.sendMsg("random_gave "+player.name+" "+card)
		player.sendMsg("random_recv "+target.name+" "+card)

		g.hands[player].addCard(card)
		player.sendMsg("hand" + g.hands[player].cardList())
		target.sendMsg("hand" + g.hands[target].cardList())
		g.favouring = nil
		g.favoured = nil
	case "steal_who":
		if g.favouring != player {
			break
		}

		target := g.playerByName(answer)
		if target == nil || target == player {
			player.sendMsg("q " + question)
			break
		}

		g.favoured = target
		player.sendMsg("q steal_what")
	case "favour_what":
		if g.favoured != player {
			player.sendMsg("err illegal_move")
			break
		}

		card, err := strconv.Atoi(answer)
		if err != nil {
			player.sendMsg("err illegal_move")
			break
		}

		if card >= g.hands[player].getLength() {
			player.sendMsg("err illegal_move")
			break
		}

		cardText := g.hands[player].getCard(card)

		// TODO: prevent favouring a nope
		// this is harder than it seems and will require some changes in the game's logic...

		g.hands[player].removeCard(card)
		player.sendMsg("hand" + g.hands[player].cardList())

		g.hands[g.favouring].addCard(cardText)
		g.favouring.sendMsg("hand" + g.hands[g.favouring].cardList())

		// The favour transaction is complete
		g.favouring.sendMsg("unlock")
		g.favouring.sendMsg("favour_recv " + g.favoured.name + " " + cardText)
		g.favoured.sendMsg("favour_gave " + g.favouring.name + " " + cardText)
		g.lobby.sendComplexBcast("favour_complete "+g.favouring.name+" "+g.favoured.name,
			map[*Client]bool{g.favoured: true, g.favouring: true})
		g.favouring = nil
		g.favoured = nil
	case "steal_what":
		if g.favouring != player || g.favoured == nil {
			break
		}

		if g.playerNumber(g.favoured) == -1 {
			// The player being asked has left :(
			player.sendMsg("q " + question)
			break
		}

		if !g.hands[g.favoured].contains(answer) {
			g.lobby.sendBcast("steal_n "+g.favouring.name+" "+g.favoured.name+" "+answer)
			break
		} else {
			g.hands[g.favoured].removeByName(answer)
			g.favoured.sendMsg("hand" + g.hands[g.favoured].cardList())

			g.hands[player].addCard(answer)
			player.sendMsg("hand" + g.hands[g.favouring].cardList())

			g.lobby.sendBcast("steal_y "+g.favouring.name+" "+g.favoured.name+" "+answer)
		}

		g.favouring = nil
		g.favoured = nil
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

	g.lobby.sendBcast("now_playing " + g.players[g.currentPlayer].name)
	if g.deck.cardsLeft() == 0 {
		g.lobby.sendBcast("draw_pile no")
	} else {
		g.lobby.sendBcast("draw_pile yes")
	}
}

func (g *Game) start() {
	// Starts the game

	g.started = true

	g.lobby.sendBcast("clear_message")
	g.lobby.sendBcast("bcast starting")

	// Shuffle the player list and re-send it
	rand.Seed(time.Now().UnixNano()) // Hard enough to predict; should be fine
	rand.Shuffle(len(g.players), func(i, j int) {
		g.players[i], g.players[j] = g.players[j], g.players[i]
	})
	g.currentPlayer = 0
	g.lobby.sendBcast("players" + g.playerList())
	g.lobby.sendBcast("now_playing " + g.players[g.currentPlayer].name)

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
