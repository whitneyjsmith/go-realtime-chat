package main

import (
	"log"
	"net/http"

	// Third party package which enables websockets in go
	"github.com/gorilla/websocket"
)

// GLOBAL VARIABLES, bad practice but we are using them anyway
var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan Message) // broadcast channel, queue for messages

// Configure the upgrader
// Takes a normal HTTP connection and upgrades it to a WebSocket
var upgrader = websocket.Upgrader{}

// Define our message object to hold messages
//
// NOTE: The text surrounded by backticks is just metadata which helps
// 		Go serialize and unserialize the Message object to and from JSON.
type Message struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

// main entry point of any Go application is always the "main()" function
func main() {
	// Create a simple file server
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)

	// Configure websocket route
	http.HandleFunc("/ws", handleConnections)

	// Start listening for incoming chat messages
	//
	// goroutines are backround processes or asynchronous functions
	go handleMessages()

	// Start the server on localhost port 8000 and log any errors
	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// handles our incoming WebSocket connections
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Make sure we close the connection when the function returns
	defer ws.Close()

	// Register our new client
	clients[ws] = true

	// An inifinite loop that continuously waits for a new message to be written to the WebSocket,
	// unserializes it from JSON to a Message object and then throws it into the broadcast channel
	//
	// If there is some kind of error with reading from the socket, we assume the client has disconnected
	// for some reason or another.
	for {
		var msg Message
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		// Send the newly received message to the broadcast channel
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		// Send it out to every client that is currently connected
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}