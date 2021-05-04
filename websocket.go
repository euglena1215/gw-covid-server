package main

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
	"github.com/k0kubun/pp"
	"github.com/labstack/echo/v4"
)

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = websocket.Upgrader{ReadBufferSize: socketBufferSize, WriteBufferSize: socketBufferSize}
var allClients = map[string][]Client{}

type Client struct {
	Ws     *websocket.Conn
	UserId string
}

type Message struct {
	Event  string `json:"event"`
	RoomId string `json:"room_id"`
	UserId string `json:"user_id"`
}

var broadcast = make(chan Message)

type RoomWebSocketRequest struct {
	UserId string `query:"user_id"`
}

func handleRoomWebsocket(c echo.Context) error {
	req := new(RoomWebSocketRequest)
	if err := c.Bind(req); err != nil {
		return err
	}
	pp.Println(req)

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	roomId := c.Param("room_id")
	client := Client{
		Ws:     ws,
		UserId: req.UserId,
	}
	allClients[roomId] = append(allClients[roomId], client)

	message := Message{
		Event:  "joinRoom",
		RoomId: roomId,
		UserId: req.UserId,
	}

	broadcast <- message

	for {
		message, err := readMessage(ws)
		if err != nil {
			c.Logger().Error(err)
		}

		switch {
		case message.Event == "game_start:AvoidYuriko":
			go setUpAvoidYuriko(message.RoomId)
			broadcast <- message
		}
	}
}

func readMessage(ws *websocket.Conn) (Message, error) {
	_, msg, err := ws.ReadMessage()
	message := new(Message)
	if err := json.Unmarshal(msg, message); err != nil {
		log.Fatal(err)
	}
	return *message, err
}

func receiveBroadCast() {
	for {
		select {
		case message := <-broadcast:
			pp.Print(message)
			clients := allClients[message.RoomId]
			for _, client := range clients {
				if client.UserId != message.UserId {
					err := client.Ws.WriteJSON(message)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}
}

func setUpAvoidYuriko(roomId string) {
	clients := allClients[roomId]

	db := connectDb()
	defer db.Close()

	for _, client := range clients {
		stmt, err := db.Prepare("INSERT INTO avoid_yuriko_users(user_id, room_id) VALUES($1,$2)")
		if err != nil {
			log.Fatal(err)
		}
		stmt.Exec(client.UserId, roomId)
	}
}
