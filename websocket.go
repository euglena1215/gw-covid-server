package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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
	Event   string `json:"event"`
	RoomId  string `json:"room_id"`
	UserId  string `json:"user_id"`
	Details string `json:"details"`
}

var broadcast = make(chan Message)

type MessageJoinRoomDetail struct {
	PlayerCount int `json:"player_count"`
}

type RoomWebSocketRequest struct {
	UserId string `query:"user_id"`
}

func handleRoomWebsocket(c echo.Context) error {
	req := new(RoomWebSocketRequest)
	if err := c.Bind(req); err != nil {
		return err
	}
	pp.Println(req)

	roomId := c.Param("room_id")
	db, err := initDb()
	if err != nil {
		return err
	}

	if exists, err := db.existsRoomById(roomId); err != nil {
		return err
	} else if !exists {
		return c.String(http.StatusNotFound, "{\"message\":\"部屋が見つかりません\"}")
	}

	if exists, err := db.existsUserByRoomIdAndUserId(roomId, req.UserId); err != nil {
		return err
	} else if !exists {
		return c.String(http.StatusBadRequest, "{\"message\":\"存在しないユーザーで入室しようとしています\"}")
	}

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	client := Client{
		Ws:     ws,
		UserId: req.UserId,
	}
	allClients[roomId] = append(allClients[roomId], client)

	details := MessageJoinRoomDetail{
		PlayerCount: len(allClients[roomId]),
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		return err
	}
	message := Message{
		Event:   "Room:Join",
		RoomId:  roomId,
		UserId:  req.UserId,
		Details: string(encoded),
	}

	broadcast <- message

	for {
		message, err := readMessage(ws)
		if err != nil {
			c.Logger().Error(err)
		}

		switch {
		case message.Event == "GameStart:AvoidYuriko":
			go func() {
				err = startAvoidYuriko(message.RoomId)
				if err != nil {
					sendSudpendEvent(err, roomId)
				}
			}()

			broadcast <- message
		case message.Event == "AvoidYuriko:AddPoint":
			go addAvoidYurikoPoint(message)
		}
	}
}

func readMessage(ws *websocket.Conn) (Message, error) {
	_, msg, err := ws.ReadMessage()
	if len(msg) == 0 {
		return Message{}, nil
	}

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
				if message.Event == "Room:Join" || client.UserId != message.UserId {
					err := client.Ws.WriteJSON(message)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}
}

type AvoidYurikoState struct {
	Remaining  float32        `json:"remaining"`
	UserScores map[string]int `json:"user_scores"`
}

func startAvoidYuriko(roomId string) error {
	clients := allClients[roomId]

	db, err := initDb()
	if err != nil {
		return err
	}
	defer db.close()

	var userIds []string
	for _, client := range clients {
		userIds = append(userIds, client.UserId)
	}

	for _, userId := range userIds {
		if err = db.storeAvoidYurikoUser(userId, roomId); err != nil {
			return err
		}
	}

	go func() {
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()

		const MAX_TIME = 30
		var time float32 = MAX_TIME

		for finished := false; !finished; {
			select {
			case <-t.C:
				if time > 0 {
					time -= 0.5

					go func() {
						err = func() error {
							db, err := initDb()
							if err != nil {
								return err
							}
							defer db.close()

							userScore := map[string]int{}
							for _, userId := range userIds {
								point, err := db.getAvoidYurikoPointByRoomIdAndUserId(roomId, userId)
								if err != nil {
									return err
								}
								userScore[userId] = point
							}

							state := AvoidYurikoState{
								Remaining:  time,
								UserScores: userScore,
							}
							encoded, err := json.Marshal(state)
							if err != nil {
								return err
							}

							message := Message{
								Event:   "AvoidYuriko:State",
								RoomId:  roomId,
								Details: string(encoded),
							}

							broadcast <- message
							return nil
						}()
						if err != nil {
							sendSudpendEvent(err, roomId)
						}
					}()
				} else {
					finished = true
					message := Message{
						Event:  "AvoidYuriko:Finish",
						RoomId: roomId,
					}

					broadcast <- message
				}
			}
		}
	}()

	return nil
}

type AddAvoidYurikoPoint struct {
	Point int `json:"point"`
}

func addAvoidYurikoPoint(message Message) error {
	var detail AddAvoidYurikoPoint
	err := json.Unmarshal([]byte(message.Details), &detail)
	if err != nil {
		log.Fatal(err)
	}

	db, err := initDb()
	if err != nil {
		return err
	}
	defer db.close()

	err = db.IncrementAvoidYurikoPoint(detail.Point, message.RoomId, message.UserId)
	if err != nil {
		return err
	}

	return nil
}

func sendSudpendEvent(err error, roomId string) {
	message := Message{
		Event:   "AvoidYuriko:Suspend",
		RoomId:  roomId,
		Details: fmt.Sprintf("{\"message\": \"%s\"}", err.Error()),
	}
	broadcast <- message
}
