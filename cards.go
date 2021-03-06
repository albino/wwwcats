package main

import (
	"math/rand"
	"time"
	"sort"
)

type Deck struct {
	// The end of the slice is the top of the deck
	cards []string
}

func newDeck() (d *Deck) {
	d = new(Deck)

	// Add all the cards, EXCEPT those which should not be dealt to players
	d.insertMultiple(map[string]int{
		"nope":    5,
		"attack":  4,
		"skip":    4,
		"favour":  4,
		"shuffle": 4,
		"see3":    5,
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
		"defuse":    6 - players,
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

func (d *Deck) peek(num int) (ret []string) {
	from := len(d.cards) - num
	if from < 0 {
		from = 0
	}

	length := num
	if len(d.cards) < num {
		length = len(d.cards)
	}

	ret = make([]string, length)
	copy(ret, d.cards[from:])

	// Reverse
	for i := len(ret)/2 - 1; i >= 0; i-- {
		opp := len(ret) - 1 - i
		ret[i], ret[opp] = ret[opp], ret[i]
	}

	return
}

func (d *Deck) shuffle() {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(d.cards), func(i, j int) {
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	})
}

func (d *Deck) dealHand(playerCount int) (h *Hand) {
	h = new(Hand)

	h.cards = []string{"defuse"}

	var cardCount int
	if playerCount > 5 {
		cardCount = 6
	} else {
		cardCount = 7
	}

	for i := 0; i < cardCount; i++ {
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

func (h *Hand) removeByName(wanted string) {
	for i, card := range h.cards {
		if card == wanted {
			h.cards = append(h.cards[:i], h.cards[i+1:]...)
			return
		}
	}
}

func (h *Hand) contains(wanted string) bool {
	for _, card := range h.cards {
		if card == wanted {
			return true
		}
	}
	return false
}

func (h *Hand) containsMultiple(wanted string, num int) bool {
	found := 0
	for _, card := range h.cards {
		if card == wanted {
			found += 1
			if found == num {
				return true
			}
		}
	}
	return false
}

func (h *Hand) getLength() int {
	return len(h.cards)
}

func (h *Hand) takeRandom() string {
	rand.Seed(time.Now().UnixNano())
	cardNo := rand.Intn(len(h.cards))
	card := h.getCard(cardNo)
	h.removeCard(cardNo)
	return card
}

func (h *Hand) sort() {
	weights := map[string]int{
		"defuse": 10,
		"nope": 20,
		"skip": 30,
		"attack": 40,
		"see3": 50,
		"shuffle": 60,
		"favour": 70,
		"random1": 80,
		"random2": 90,
		"random3": 100,
		"random4": 110,
		"random5": 120,
	}

	sort.Slice(h.cards, func (a, b int) bool {
		return weights[h.cards[a]] < weights[h.cards[b]]
	})
}

type GameState struct {
	// Used to save the game state before performing a NOPEable action
	// needs to contain anything that can be reversed by a NOPE
	deck          Deck
	currentPlayer int
	attack        bool
}

func makeGameState(g *Game) *GameState {
	// Copy everything
	cards := make([]string, len(g.deck.cards))
	copy(cards, g.deck.cards)

	return &GameState{
		deck: Deck{
			cards: cards,
		},
		currentPlayer: g.currentPlayer,
		attack:        g.attack,
	}
}

func (gs *GameState) restore(g *Game) {
	g.deck = &gs.deck
	g.currentPlayer = gs.currentPlayer
	g.attack = gs.attack
}
