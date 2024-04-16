package main

import (
	"fmt"
	"github.com/Baiguoshuai1/shadiaosocketio"
	"github.com/Baiguoshuai1/shadiaosocketio/websocket"
	"log"
	"net/http"
	"time"
)

type Message struct {
	Id      int    `json:"id"`
	Channel string `json:"channel"`
}

type Desc struct {
	Text string `json:"text"`
}

func main() {
	server := shadiaosocketio.NewServer(*websocket.GetDefaultWebsocketTransport())

	server.On(shadiaosocketio.OnConnection, func(c *shadiaosocketio.Channel) {
		logWithTimestamp("connected! id:", c.Id(), c.LocalAddr().Network()+" "+c.LocalAddr().String()+
			" --> "+c.RemoteAddr().Network()+" "+c.RemoteAddr().String())

		//c.Join("room")
		//server.BroadcastTo("room", "/admin", Message{1, "new members!"})
		//time.Sleep(100 * time.Millisecond)
		//c.BroadcastTo("room", "/admin", Message{2, "hello everyone!"})

		_ = c.Emit("message", Message{1, "{\"chinese\":\"中文才是最屌的\"}"})

		time.Sleep(1 * time.Second)
		logWithTimestamp("send method: ackFromServer, wait for 2s")
		result, err := c.Ack("/ackFromServer", 5*time.Second, "server", 2)
		if err != nil {
			logWithTimestamp("on ackFromServer cb err:", err)
		} else {
			if c.BinaryMessage() {
				logWithTimestamp("on ackFromServer cb raw:", result)
				return
			}

			// [][]byte
			res := result.([]interface{})
			logWithTimestamp("on ackFromServer cb raw buffer:", result)
			logWithTimestamp("on ackFromServer cb stringify:", string(res[0].([]byte)), string(res[1].([]byte)))
		}
	})

	server.On(shadiaosocketio.OnDisconnection, func(c *shadiaosocketio.Channel, reason websocket.CloseError) {
		logWithTimestamp("disconnect", c.Id(), "code:", reason.Code, "text:", reason.Text)
	})

	server.On("message", func(c *shadiaosocketio.Channel, arg1 string, arg2 Message, arg3 int, arg4 bool) {
		logWithTimestamp("on message:", "arg1:", arg1, "arg2:", arg2, "arg3:", arg3, "arg4:", arg4)
	})

	server.On("/ackFromClient", func(c *shadiaosocketio.Channel, msg Message, num int) (int, Desc, string) {
		logWithTimestamp("on ackFromClient:", msg, num)
		time.Sleep(3 * time.Second)
		return 3, Desc{Text: "ackFromServer"}, "string"
	})

	serveMux := http.NewServeMux()
	serveMux.Handle("/socket.io/", server)

	logWithTimestamp("running...")
	log.Panic(http.ListenAndServe(":2233", serveMux))
}

func logWithTimestamp(args ...interface{}) {
	timestamp := time.Now()
	formattedTime := fmt.Sprintf("%02d:%02d:%02d.%03d", timestamp.Hour(), timestamp.Minute(), timestamp.Second(), timestamp.Nanosecond()/int(time.Millisecond))
	fmt.Printf("%s [server] ", formattedTime)
	fmt.Println(args...)
}
