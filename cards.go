package main

import (
	"math/rand"
	"time"
)

type Deck struct {
	// The end of the slice is the top of the deck
	cards []string
}

func newDeck() (d *Deck) {
	d = new(Deck)

	// Add all the cards, EXCEPT those which should not be dealt to players
	d.insertMultiple(map[string]int{
		"nope": 5,
		"attack": 4,
		"skip": 4,
		"favour": 4,
		"shuffle": 4,
		"see3": 5,
		"random1": 4,
		"random2": 4,
		"random3": 4,
		"random4": 4,
		"random5": 4,
	})

	d.shuffle()

	return
}

func (d *Deck) addExtraCards(players int) {
	d.insertMultiple(map[string]int{
		"exploding": players - 1,
		"defuse": 6 - players,
	})

	d.shuffle()

	return
}

func (d *Deck) insertOnTop(card string) {
	d.cards = append(d.cards, card)
}

func (d *Deck) insertMultiple(cards map[string]int) {
	for card, number := range cards {
		for i := 0; i < number; i++ {
			d.insertOnTop(card)
		}
	}
}

func (d *Deck) insertAtPos(pos int, card string) {
	// the top of the deck is the end of the array
	pos = len(d.cards) - (pos)
	d.cards = append(d.cards, "")
	copy(d.cards[pos+1:], d.cards[pos:])
	d.cards[pos] = card
}

func (d *Deck) draw() (card string) {
	// pop
	card, d.cards = d.cards[len(d.cards)-1], d.cards[:len(d.cards)-1]
	return
}

func (d *Deck) shuffle() {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(d.cards), func (i, j int) {
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	})
}

func (d *Deck) dealHand() (h *Hand) {
	h = new(Hand)

	h.cards = []string{"defuse"}

	for i := 0; i < 7; i++ {
		h.cards = append(h.cards, d.draw())
	}

	return
}

func (d *Deck) cardsLeft() int {
	return len(d.cards)
}

type Hand struct {
	cards []string
}

func (h *Hand) cardList() (list string) {
	// TODO sorting

	for _, card := range h.cards {
		list = list + " " + card
	}

	return
}

func (h *Hand) getCard(num int) string {
	return h.cards[num]
}

func (h *Hand) addCard(card string) {
	h.cards = append(h.cards, card)
}

func (h *Hand) removeCard(num int) {
	h.cards = append(h.cards[:num], h.cards[num+1:]...)
}

func (h *Hand) contains(wanted string) bool {
	for _, card := range h.cards {
		if card == wanted {
			return true
		}
	}
	return false
}

func (h *Hand) getLength() int {
	return len(h.cards)
}

type GameState struct {
	// Used to save the game state before performing a NOPEable action
	// needs to contain anything that can be reversed by a NOPE
	deck Deck
	currentPlayer int
	attack bool
}

func makeGameState(g *Game) *GameState {
	// Copy everything
	cards := make([]string, len(g.deck.cards))
	copy(cards, g.deck.cards)

	return &GameState {
		deck: Deck {
			cards: cards,
		},
		currentPlayer: g.currentPlayer,
		attack: g.attack,
	}
}

func (gs *GameState) restore(g *Game) {
	g.deck = &gs.deck
	g.currentPlayer = gs.currentPlayer
	g.attack = gs.attack
}
