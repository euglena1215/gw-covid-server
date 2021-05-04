package main

import (
	"fmt"
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
var allClients = map[string][]*websocket.Conn{}

type Message struct {
	Event  string `json:"event"`
	RoomId string `json:"room_id"`
	UserId string `json:"user_id"`
}

var broadcast = make(chan Message)

func handleRoomWebsocket(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	roomId := c.Param("room_id")
	allClients[roomId] = append(allClients[roomId], ws)
	pp.Println(roomId)
	pp.Println(ws)
	pp.Println(allClients)

	message := Message{
		Event:  "joinRoom",
		RoomId: roomId,
		UserId: "xxxxxx", // あとで Authorization Header から取得する
	}

	pp.Print(message)

	broadcast <- message

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			c.Logger().Error(err)
		}
		fmt.Printf("%s\n", msg)
	}
}

func receiveBroadCast() {
	for {
		select {
		case message := <-broadcast:
			pp.Print(message)
			clients := allClients[message.RoomId]
			for _, client := range clients {
				err := client.WriteJSON(message)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}
