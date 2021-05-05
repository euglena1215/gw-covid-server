package main

import (
	"database/sql"
	"os"
)

type Db struct {
	conn *sql.DB
}

func initDb() (Db, error) {
	conn, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return Db{}, err
	}

	db := Db{conn: conn}
	return db, nil
}

func (db Db) close() {
	db.conn.Close()
}

func (db Db) storeUser(userId string, roomId string) error {
	stmt, err := db.conn.Prepare("INSERT INTO users(id, room_id) VALUES($1,$2)")
	if err != nil {
		return err
	}
	stmt.Exec(userId, roomId)

	return nil
}

func (db Db) storeRoom(roomId string) error {
	stmt, err := db.conn.Prepare("INSERT INTO rooms(id) VALUES($1)")
	if err != nil {
		return err
	}
	stmt.Exec(roomId)

	return nil
}

func (db Db) storeAvoidYurikoUser(userId string, roomId string) error {
	stmt, err := db.conn.Prepare("INSERT INTO avoid_yuriko_users(user_id, room_id) VALUES($1,$2)")
	if err != nil {
		return err
	}
	stmt.Exec(userId, roomId)

	return nil
}

func (db Db) existsRoomById(roomId string) (bool, error) {
	var exists int
	err := db.conn.QueryRow("SELECT 1 FROM rooms WHERE id = $1", roomId).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists == 1, nil
}

func (db Db) getAvoidYurikoPointByRoomIdAndUserId(roomId string, userId string) (int, error) {
	var point int
	err := db.conn.QueryRow("SELECT point FROM avoid_yuriko_users WHERE room_id = $1 AND user_id = $2", roomId, userId).Scan(&point)
	if err != nil {
		return 0, err
	}
	return point, nil
}
