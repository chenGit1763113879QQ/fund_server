package model

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
)

type Conn struct {
	sync.Mutex
	Conn *websocket.Conn
	Err  error
}

var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	EnableCompression: true,
}

func NewWebSocket(c *gin.Context) *Conn {
	ws := new(Conn)
	ws.Conn, ws.Err = upGrader.Upgrade(c.Writer, c.Request, nil)
	go ws.Ping()
	return ws
}

func (c *Conn) WriteJson(data any) {
	if c.Err != nil {
		return
	}
	c.Lock()
	defer c.Unlock()

	c.Err = c.Conn.WriteJSON(data)

}

func (c *Conn) WriteBson(src any) {
	if c.Err != nil {
		return
	}
	c.Lock()
	defer c.Unlock()

	var dst bson.M
	str, _ := bson.Marshal(src)
	bson.Unmarshal(str, &dst)
	c.Err = c.Conn.WriteJSON(dst)
}

func (c *Conn) ReadJson(data any) {
	c.Err = c.Conn.ReadJSON(&data)
}

func (c *Conn) Ping() {
	for c.Alive() {
		time.Sleep(time.Second * 10)
		c.WriteJson(map[string]string{
			"type": "ping", "time": time.Now().Format("2006/01/02 15:04:05"),
		})
	}
}

func (c *Conn) Alive() bool {
	return c != nil && c.Err == nil
}
