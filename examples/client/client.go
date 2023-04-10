package main

import (
	"encoding/json"
	"github.com/Baiguoshuai1/shadiaosocketio"
	"github.com/Baiguoshuai1/shadiaosocketio/websocket"
	"github.com/buger/jsonparser"
	"log"
	"time"
)

type Message struct {
	Id      int    `json:"id"`
	Channel string `json:"channel"`
}

type Desc struct {
	Text string `json:"text"`
}

func sendAck(c *shadiaosocketio.Client) {
	// return [][]byte
	result, err := c.Ack("/ackFromClient", time.Second*5, Message{Id: 3, Channel: "client channel"}, 4)
	if err != nil {
		log.Println("[client] ack cb err:", err)
	} else {
		res := result.([]interface{})

		if c.BinaryMessage() {
			log.Println("[client] ack cb:", res)
			return
		}

		if len(result.([]interface{})) == 0 {
			return
		}
		var outArg1 int
		var outArg2 Desc
		var outArg3 string

		err := json.Unmarshal(res[0].([]byte), &outArg1)
		if err != nil {
			log.Println("[client] ack cb err:", err)
			return
		}
		log.Println("[client] ack cb outArg1:", outArg1)

		err = json.Unmarshal(res[1].([]byte), &outArg2)
		if err != nil {
			log.Println("[client] ack cb err:", err)
			return
		}
		log.Println("[client] ack cb outArg2:", outArg2.Text)

		err = json.Unmarshal(res[2].([]byte), &outArg3)
		if err != nil {
			log.Println("[client] ack cb err:", err)
			return
		}
		log.Println("[client] ack cb outArg3:", outArg3)
	}
}

func sendMessage(c *shadiaosocketio.Client, args ...interface{}) {
	err := c.Emit("message", args...)
	if err != nil {
		panic(err)
	}
}

func createClient() *shadiaosocketio.Client {
	c, err := shadiaosocketio.Dial(
		shadiaosocketio.GetUrl("localhost", 2233, false),
		*websocket.GetDefaultWebsocketTransport())
	if err != nil {
		panic(err)
	}

	_ = c.On(shadiaosocketio.OnConnection, func(h *shadiaosocketio.Channel) {
		log.Println("[client] connected! id:", h.Id())
		log.Println("[client]", h.LocalAddr().Network()+" "+h.LocalAddr().String()+
			" --> "+h.RemoteAddr().Network()+" "+h.RemoteAddr().String())
	})
	_ = c.On(shadiaosocketio.OnDisconnection, func(h *shadiaosocketio.Channel, reason websocket.CloseError) {
		log.Println("[client] disconnected, code:", reason.Code, "text:", reason.Text)
	})

	_ = c.On("message", func(h *shadiaosocketio.Channel, args Message) {
		str, err := jsonparser.GetString([]byte(args.Channel), "chinese")
		if err != nil {
			log.Println("[client] parse json err:", err)
			return
		}
		log.Println("[client] got chat message:", str)
	})
	_ = c.On("/admin", func(h *shadiaosocketio.Channel, args Message) {
		log.Println("[client] got admin message:", args)
	})
	// sending ack response
	_ = c.On("/ackFromServer", func(h *shadiaosocketio.Channel, arg1 string, arg2 int) (Message, int) {
		log.Println("[client] got ack from server:", arg1, arg2)
		time.Sleep(2 * time.Second)
		return Message{
			Id:      5,
			Channel: "client channel",
		}, 6
	})

	return c
}

func main() {
	c := createClient()

	time.Sleep(1 * time.Second)
	sendMessage(c, "client", &Message{
		Id:      1,
		Channel: "client channel",
	}, 2)

	time.Sleep(1 * time.Second)
	sendAck(c)

	time.Sleep(3 * time.Second)
	log.Println("ReadBytes length:", c.ReadBytes())
	log.Println("WriteBytes length:", c.WriteBytes())

	select {}
}
