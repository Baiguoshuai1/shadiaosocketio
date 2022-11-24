package main

import (
	"encoding/json"
	"github.com/Baiguoshuai1/shadiaosocketio"
	"github.com/Baiguoshuai1/shadiaosocketio/websocket"
	"log"
	"time"
)

type Channel struct {
	Channel string `json:"channel"`
}

type Message struct {
	Id      int    `json:"id"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func sendJoin(c *shadiaosocketio.Client) {
	// return [][]byte
	result, err := c.Ack("/admin", time.Second*5, Channel{"admin"})
	if err != nil {
		log.Println("sendJoin cb err:", err)
	} else {
		if len(result.([]interface{})) == 0 {
			return
		}

		var outArg1 int
		var outArg2 string

		err := json.Unmarshal(result.([]interface{})[0].([]byte), &outArg1)
		if err != nil {
			log.Println("sendJoin cb err:", err)
			return
		}
		err = json.Unmarshal(result.([]interface{})[1].([]byte), &outArg2)
		if err != nil {
			log.Println("sendJoin cb err:", err)
			return
		}

		log.Println("sendJoin cb:", outArg1, outArg2)
	}
}

func sendMsg(c *shadiaosocketio.Client, args ...interface{}) {
	err := c.Emit("message", args...)
	if err != nil {
		panic(err)
	}
}

func createClient() {
	c, err := shadiaosocketio.Dial(
		shadiaosocketio.GetUrl("localhost", 2233, false),
		*websocket.GetDefaultWebsocketTransport())
	if err != nil {
		panic(err)
	}

	err = c.On("message", func(h *shadiaosocketio.Channel, args Message) {
		log.Println("--- Got chat message: ", args)
	})
	if err != nil {
		panic(err)
	}

	err = c.On("/admin", func(h *shadiaosocketio.Channel, args Message) {
		log.Println("--- Got admin message: ", args)
	})
	if err != nil {
		panic(err)
	}

	err = c.On(shadiaosocketio.OnDisconnection, func(h *shadiaosocketio.Channel, reason websocket.CloseError) {
		log.Println("Disconnected, code:", reason.Code, "text:", reason.Text)
	})
	if err != nil {
		panic(err)
	}

	err = c.On(shadiaosocketio.OnConnection, func(h *shadiaosocketio.Channel) {
		log.Println("Connected! id:", h.Id())
		log.Println(h.LocalAddr().Network() + " " + h.LocalAddr().String() +
			" --> " + h.RemoteAddr().Network() + " " + h.RemoteAddr().String())
	})

	time.Sleep(1 * time.Second)

	sendMsg(c, "cool", &Message{
		Id:   99,
		Text: "second arg",
	})
	sendJoin(c)
	if err != nil {
		panic(err)
	}
}

func main() {
	createClient()

	select {}
}
