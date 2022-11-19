package main

import (
	"github.com/Baiguoshuai1/shadiaosocketio"
	"github.com/Baiguoshuai1/shadiaosocketio/websocket"
	"log"
	"reflect"
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
	result, err := c.Ack("/admin", time.Second*5, Channel{"admin"})
	if err != nil {
		panic(err)
	} else {
		log.Println("cb:", result, reflect.TypeOf(result))
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

	err = c.On("/message", func(h *shadiaosocketio.Channel, args Message) {
		log.Println("--- Got chat message: ", args)
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
		log.Println("Connected!")
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
