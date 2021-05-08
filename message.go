package main

import (
	"encoding/json"
	"errors"
)

//
// websocket Message 全体設計
//
// イベントは送信・受信ともに event, room_id, user_id, details の組で表現される。
// データ形式は JSON で、 details フィールドには追加情報が JSON の文字列で格納されている場合がある。
// 例
// {
//   "event":"AvoidYuriko:State",
//   "room_id":"f6d7ce7e-abfa-11eb-aa83-acde48001122",
//   "user_id":  "",
//   "details": "{\"remaining\":3.5,\"user_scores\":{\"aaaaa\":0,\"bbbbbb\":0}}"
// }
//

const (
	// 部屋関連
	EVENT_ROOM_JOIN string = "Room:Join"

	// ゲーム全般
	EVENT_GAME_START_AVOID_YURIKO string = "GameStart:AvoidYuriko"

	// ゆりこ
	EVENT_AVOID_YURIKO_ADD_POINT string = "AvoidYuriko:AddPoint"
	EVENT_AVOID_YURIKO_STATE     string = "AvoidYuriko:State"
	EVENT_AVOID_YURIKO_SUSPEND   string = "AvoidYuriko:Suspend"
	EVENT_AVOID_YURIKO_FINISH    string = "AvoidYuriko:Finish"
)

type Message struct {
	Event   string `json:"event"`
	RoomId  string `json:"room_id"`
	UserId  string `json:"user_id"`
	Details string `json:"details"`
}

type RoomJoinDetail struct {
	PlayerCount int `json:"player_count"`
}

type AvoidYurikoStateDetail struct {
	Remaining  float32        `json:"remaining"`
	UserScores map[string]int `json:"user_scores"`
}

type AvoidYurikoAddPointDetail struct {
	Point int `json:"point"`
}

var broadcast = make(chan Message, 100)

func (message *Message) setDetail(payload interface{}) (Message, error) {
	// 型の動的チェック
	switch message.Event {
	case EVENT_ROOM_JOIN:
		_ = payload.(RoomJoinDetail)
	case EVENT_AVOID_YURIKO_STATE:
		_ = payload.(AvoidYurikoStateDetail)
	case EVENT_AVOID_YURIKO_ADD_POINT:
		_ = payload.(AvoidYurikoAddPointDetail)
	default:
		panic(errors.New("そのメッセージには details は存在しません"))
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}

	message.Details = string(encoded)
	return *message, nil
}
