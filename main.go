package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// "*" means a pointer of type "a".  "&" returns the pointer of a
// variable.  &type doesn't make sense; &instance does.
var clients = make(map[*websocket.Conn]struct{})
var channel = make(chan Message)  // this accepts and returns type Message
var socket = websocket.Upgrader{} // curlies initialize a struct

type Message struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}

func main() {
	// Define our application's routes.
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)
	http.HandleFunc("/ws", handleConnections)

	// Set off a goroutine to manage client messages.
	go handleMessages()

	// Start the server.
	log.Println("Server started on port 5000")
	err := http.ListenAndServe(":5000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

	// I'm blocked!
	println("You can't see me!")
}

func handleConnections(response http.ResponseWriter, request *http.Request) {
	// Debugging.
	// socket.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := socket.Upgrade(response, request, nil)
	if err != nil {
		log.Fatal("Request failed... ", err)
	}
	defer ws.Close()

	// Register our client in the global.  Apparently hashmaps are the
	// way to go in Go.  "true" seems to be idiomatic but struct{}
	// consumes 0 bytes of memory.
	clients[ws] = struct{}{}

	// Infinite loop.  I assume this is non-blocking somewhere in the
	// HTTP package, otherwise what good is it.  But I haven't
	// verified.  It works of course which leads me to believe there's
	// a goroutine being scheduled implicitly.
	for {
		// Declare a msg variable of type "Message"
		var msg Message

		// Read from the socket (blocking).
		err := ws.ReadJSON(&msg)

		// If we couldn't read the message kick the client and kill the loop.
		if err != nil {
			delete(clients, ws)
			break
		}

		// Send the message to the channel.
		channel <- msg
	}
}

func handleMessages() {
	// Infinite loop.
	for {
		// 	Await message.  This is blocking so the loop pauses here
		// until a message is sent and processed.  We're in a
		// goroutine so the rest of the application is free to
		// continue.
		msg := <-channel

		// Iterate over the connected clients...
		for client := range clients {
			// Serialize the message to JSON which coincidentally sends
			// it to the socket.
			err := client.WriteJSON(msg)

			// If the client is no longer connected remove it from the
			// global.
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
	}
}
