package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
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
	db := connectDb()
	defer db.Close()

	roomId, _ := uuid.NewUUID()
	roomStmt, roomErr := db.Prepare("INSERT INTO rooms(id) VALUES($1)")
	if roomErr != nil {
		return roomErr
	}
	roomStmt.Exec(roomId.String())

	userId, _ := uuid.NewUUID()
	userStmt, userErr := db.Prepare("INSERT INTO users(id, room_id) VALUES($1,$2)")
	if userErr != nil {
		return userErr
	}
	userStmt.Exec(userId.String(), roomId.String())

	res := CreateRoomResponse{
		RoomId: roomId.String(),
		UserId: userId.String(),
	}

	return c.JSON(http.StatusOK, res)
}

type JoinRoomRequest struct {
	RoomId string `json:"room_id"`
}

type JoinRoomResponse struct {
	UserId string `json:"user_id"`
}

func handleJoinRoom(c echo.Context) error {
	req := new(JoinRoomRequest)
	if err := c.Bind(req); err != nil {
		return err
	}

	db := connectDb()
	defer db.Close()

	var exists int
	err := db.QueryRow("SELECT 1 FROM rooms WHERE id = $1", req.RoomId).Scan(&exists)
	if err != nil {
		// 対応するレコードが見つからなかった場合もここにくる
		return c.String(http.StatusBadRequest, "{}")
	}

	userId, _ := uuid.NewUUID()
	userStmt, userErr := db.Prepare("INSERT INTO users(id, room_id) VALUES($1,$2)")
	if userErr != nil {
		return userErr
	}
	userStmt.Exec(userId.String(), req.RoomId)
	
	res := JoinRoomResponse{
		UserId: userId.String(),
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

func connectDb() *sql.DB {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.GET("/ping", handlePing)
	e.POST("/room", handleCreateRoom)
	e.POST("/room/join", handleJoinRoom)
	e.GET("/ws", handleWebSocket)
	e.Static("/", "public")
	e.Logger.Fatal(e.Start(":" + os.Getenv("PORT")))
}
