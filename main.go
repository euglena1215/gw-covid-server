package main

import (
	"os"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/websocket"
)

func handlePing(c echo.Context) error {
	return c.String(http.StatusOK, "pong")
}

type CreateRoomResponse struct {
	RoomId string `json:"room_id"`
	UserId string `json:"user_id"`
}

func handleCreateRoom(c echo.Context) error {
	res := CreateRoomResponse {
		RoomId: "xxxxx",
		UserId: "xxxxxx",
	}

	return c.JSON(http.StatusOK, res)
}

func handleWebSocket(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()

		// 初回のメッセージを送信
		err := websocket.Message.Send(ws, "Server: Hello, Client!")
		if err != nil {
			c.Logger().Error(err)
		}

		for {
			// Client からのメッセージを読み込む
			msg := ""
			err = websocket.Message.Receive(ws, &msg)
			if err != nil {
				c.Logger().Error(err)
			}

			// Client からのメッセージを元に返すメッセージを作成し送信する
			err := websocket.Message.Send(ws, fmt.Sprintf("Server: \"%s\" received!", msg))
			if err != nil {
				c.Logger().Error(err)
			}
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.GET("/ping", handlePing)
	e.POST("/room", handleCreateRoom)
	e.GET("/ws", handleWebSocket)
	e.Static("/", "public")
	e.Logger.Fatal(e.Start(":"+os.Getenv("PORT")))
}
