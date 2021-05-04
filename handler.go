package main

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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
