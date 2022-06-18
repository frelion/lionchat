package component

import (
	"github.com/gorilla/websocket"
)

type SafeWebsocket struct {
	conn *websocket.Conn
}

func (Sw *SafeWebsocket) ReadJson(v interface{}) error {
	defer func() {
		recover()
	}()
	return Sw.conn.ReadJSON(&v)
}

func (Sw *SafeWebsocket) WriteJson(v interface{}) error {
	defer func() {
		recover()
	}()
	return Sw.conn.WriteJSON(v)
}
