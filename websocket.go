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
	defer func() {
		// 対応する ws を allClients からpopする
		allClients[roomId] = func(clients []Client) []Client {
			reject := func(client Client, userId string) bool {
				return client.UserId != userId
			}

			ans := make([]Client, 0)
			for _, client := range clients {
				if reject(client, req.UserId) {
					ans = append(ans, client)
				}
			}
			return ans
		}(allClients[roomId])

		ws.Close()
	}()

	client := Client{
		Ws:     ws,
		UserId: req.UserId,
	}
	allClients[roomId] = append(allClients[roomId], client)

	message := Message{
		Event:  EVENT_ROOM_JOIN,
		RoomId: roomId,
		UserId: req.UserId,
	}

	details := RoomJoinDetail{
		PlayerCount: len(allClients[roomId]),
	}
	message, err = message.setDetail(details)
	if err != nil {
		return err
	}

	broadcast <- message

	for {
		message, err := readMessage(ws)
		if err != nil {
			c.Logger().Error(err)
			return err
		}
		println("receive")
		pp.Println(message)

		switch {
		case message.Event == EVENT_GAME_START_AVOID_YURIKO:
			go func() {
				err = startAvoidYuriko(message.RoomId)
				if err != nil {
					sendSudpendEvent(err, roomId)
				}
			}()

			broadcast <- message
		case message.Event == EVENT_AVOID_YURIKO_ADD_POINT:
			go func() error {
				err = addAvoidYurikoPoint(message)
				if err != nil {
					c.Logger().Error(err)
					return err
				}
				return nil
			}()
		}
	}
}

func readMessage(ws *websocket.Conn) (Message, error) {
	_, msg, err := ws.ReadMessage()
	if err != nil {
		return Message{}, err
	}
	if len(msg) == 0 {
		return Message{}, nil
	}

	message := new(Message)
	if err := json.Unmarshal(msg, message); err != nil {
		return Message{}, err
	}

	return *message, nil
}

func receiveBroadCast() {
	for {
		select {
		case message := <-broadcast:
			println("send")
			pp.Print(message)
			clients := allClients[message.RoomId]
			for _, client := range clients {
				if message.Event == EVENT_ROOM_JOIN || client.UserId != message.UserId {
					err := client.Ws.WriteJSON(message)
					if err != nil {
						log.Println(err)
					}
				}
			}
		}
	}
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

		const MAX_TIME = 35
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

							message := Message{
								Event:  EVENT_AVOID_YURIKO_STATE,
								RoomId: roomId,
							}
							state := AvoidYurikoStateDetail{
								Remaining:  time,
								UserScores: userScore,
							}

							message, err = message.setDetail(state)
							if err != nil {
								return err
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
						Event:  EVENT_AVOID_YURIKO_FINISH,
						RoomId: roomId,
					}

					broadcast <- message
				}
			}
		}
	}()

	return nil
}

func addAvoidYurikoPoint(message Message) error {
	var detail AvoidYurikoAddPointDetail
	err := json.Unmarshal([]byte(message.Details), &detail)
	if err != nil {
		return err
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
		Event:   EVENT_AVOID_YURIKO_SUSPEND,
		RoomId:  roomId,
		Details: fmt.Sprintf("{\"message\": \"%s\"}", err.Error()),
	}
	broadcast <- message
}
