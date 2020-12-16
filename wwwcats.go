package main

import (
	"log"
	"net/http"
	"flag"

	"github.com/gorilla/websocket"
)

var addr = flag.String("l", ":8080", "http service address")

func main() {
	flag.Parse()

	// Serve the client
	fs := http.FileServer(http.Dir("public_html"))
	http.Handle("/", fs)

	// Handle incoming websocket connections
	http.HandleFunc("/ws", handleConnections)

	// Start the server
	log.Println("Now listening on", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var upgrader = websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
}

// Upgrade incoming connections to websockets
func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Instantiate the new client object
	client := &Client{conn: conn, send: make(chan []byte, 256)}

	// Hand the client off to these goroutines which will handle all i/o
	go client.readPump()
	go client.writePump()
}
