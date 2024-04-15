package main

import (
	"encoding/json"
	"fmt"
	"github.com/Baiguoshuai1/shadiaosocketio"
	"github.com/Baiguoshuai1/shadiaosocketio/websocket"
	"github.com/buger/jsonparser"
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
	logWithTimestamp("send method: ackFromClient, wait for 3s")

	// return [][]byte
	result, err := c.Ack("/ackFromClient", time.Second*5, Message{Id: 2, Channel: "client channel"}, 2)
	if err != nil {
		logWithTimestamp("on ackFromClient cb err:", err)
	} else {
		res := result.([]interface{})

		if c.BinaryMessage() {
			logWithTimestamp("on ackFromClient cb:", res)
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
			logWithTimestamp("on ackFromClient cb err:", err)
			return
		}
		err = json.Unmarshal(res[1].([]byte), &outArg2)
		if err != nil {
			logWithTimestamp("on ackFromClient cb err:", err)
			return
		}
		err = json.Unmarshal(res[2].([]byte), &outArg3)
		if err != nil {
			logWithTimestamp("on ackFromClient cb err:", err)
			return
		}
		logWithTimestamp("on ackFromClient cb:", outArg1, outArg2.Text, outArg3)
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
		logWithTimestamp("connected! id:", h.Id(), h.LocalAddr().Network()+" "+h.LocalAddr().String()+
			" --> "+h.RemoteAddr().Network()+" "+h.RemoteAddr().String())
	})

	_ = c.On(shadiaosocketio.OnDisconnection, func(h *shadiaosocketio.Channel, reason websocket.CloseError) {
		logWithTimestamp("disconnected, code:", reason.Code, "text:", reason.Text)
	})

	_ = c.On("message", func(h *shadiaosocketio.Channel, args Message) {
		str, err := jsonparser.GetString([]byte(args.Channel), "chinese")
		if err != nil {
			logWithTimestamp("parse json err:", err)
			return
		}
		logWithTimestamp("on message:", "id:", args.Id, "channel.chinese:", str)
	})

	_ = c.On("/admin", func(h *shadiaosocketio.Channel, args Message) {
		logWithTimestamp("on admin message:", args)
	})

	_ = c.On("/ackFromServer", func(h *shadiaosocketio.Channel, arg1 string, arg2 int) (Message, int) {
		logWithTimestamp("on ackFromServer:", arg1, arg2)
		time.Sleep(2 * time.Second)
		return Message{
			Id:      3,
			Channel: "client channel",
		}, 3
	})

	return c
}

func main() {
	c := createClient()

	time.Sleep(1 * time.Second)
	sendMessage(c, "client", &Message{
		Id:      1,
		Channel: "client channel",
	}, 1)

	time.Sleep(1 * time.Second)
	sendAck(c)

	select {}
}

func logWithTimestamp(args ...interface{}) {
	timestamp := time.Now()
	formattedTime := fmt.Sprintf("%02d:%02d:%02d.%03d", timestamp.Hour(), timestamp.Minute(), timestamp.Second(), timestamp.Nanosecond()/int(time.Millisecond))
	fmt.Printf("%s [client] ", formattedTime)
	fmt.Println(args...)
}
