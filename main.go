package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

func connectDb() *sql.DB {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func main() {
	go receiveBroadCast()

	e := echo.New()
	e.Use(middleware.Logger())
	e.GET("/ping", handlePing)
	e.POST("/room", handleCreateRoom)
	e.POST("/room/join", handleJoinRoom)
	e.GET("/ws/room/:room_id", handleRoomWebsocket)
	e.Logger.Fatal(e.Start(":" + os.Getenv("PORT")))
}
