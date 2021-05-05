package main

import (
	"database/sql"
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
	db, err := initDb()
	if err != nil {
		return err
	}
	defer db.close()

	roomUuid, _ := uuid.NewUUID()
	roomId := roomUuid.String()
	userUuid, _ := uuid.NewUUID()
	userId := userUuid.String()

	err = db.storeRoom(roomId)
	if err != nil {
		return err
	}

	err = db.storeUser(userId, roomId)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, CreateRoomResponse{
		RoomId: roomId,
		UserId: userId,
	})
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
		return c.String(http.StatusInternalServerError, err.Error())
	}

	db, err := initDb()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	defer db.close()

	_, err = db.existsRoomById(req.RoomId)
	if err == sql.ErrNoRows {
		return c.String(http.StatusNotFound, "{}")
	} else if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	userUuid, _ := uuid.NewUUID()
	userId := userUuid.String()

	err = db.storeUser(userId, req.RoomId)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, JoinRoomResponse{
		UserId: userId,
	})
}
