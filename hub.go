
   
// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	// "log"
	"bytes"
	"strconv"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan *Response

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

var id = 1
var id_map = make(map[int]*Client)

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan *Response),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
			case client := <-h.register:
				// no mutex, just for testing
				client.setID(id)
				id_map[id] = client
				h.clients[client] = true

				response := &Response{id: id, cmd: "user_add"}

				for cli_id := range id_map {
					if cli_id == id {
						continue;
					}
					client.send <- &Response{id: cli_id, cmd: "user_add"}
				}

				for cli := range h.clients {
					if cli.id == id {
						continue;
					}
					select {
						case cli.send <- response:
						default:
							close(cli.send)
							delete(h.clients, cli)
					}
				}

				id++

			case client := <-h.unregister:
				if _, ok := h.clients[client]; ok {
					id := client.id
					delete(id_map, client.id)
					delete(h.clients, client)
					close(client.send)

					for cli := range h.clients {
						select {
							case cli.send <- &Response{id: id, cmd: "user_delete"}:
							default:
								close(cli.send)
								delete(h.clients, cli)
						}
					}
				}
			case response := <-h.broadcast:
				res := bytes.Split(response.message, []byte("-"))
				cmd, message := res[0], res[1]

				if string(cmd[:]) == "0" {
					response.message = message
					for client := range h.clients {
						select {
							case client.send <- response:
							default:
								close(client.send)
								delete(h.clients, client)
						}
					}
				} else {
					id, _ := strconv.Atoi(string(cmd[:]))

					if client, ok := id_map[id]; ok {
						response := &Response{
							id: response.id, 
							cmd: "murmur", 
							message: message,
						}
						
						client.send <- response

						if selfClient, ok := id_map[response.id]; ok {
							selfClient.send <- response
						}
					}
				}
			}
	}
}