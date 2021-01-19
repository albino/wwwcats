package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var addr = flag.String("l", ":8080", "http service address")

var REVISION = 3

func main() {
	flag.Parse()

	// Create a global list of lobbies
	lobbies := make(map[string]*Lobby)

	// Serve the client-side software
	fs := http.FileServer(http.Dir("public_html"))
	http.Handle("/", fs)

	// Handle incoming websocket connections
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleConnections(w, r, lobbies)
	})

	// Start the server
	log.Println("Now listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Upgrade incoming connections to websockets
func handleConnections(w http.ResponseWriter, r *http.Request, lobbies map[string]*Lobby) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Instantiate the new client object
	client := &Client{conn: conn, send: make(chan []byte, 256)}

	// Hand the client off to these goroutines which will handle all i/o
	go client.readPump(lobbies)
	go client.writePump()
}
