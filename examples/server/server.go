package main

import (
	"github.com/Baiguoshuai1/shadiaosocketio"
	"github.com/Baiguoshuai1/shadiaosocketio/websocket"
	"log"
	"net/http"
)

type Channel struct {
	Channel string `json:"channel"`
}

type Message struct {
	Id      int    `json:"id"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func main() {
	server := shadiaosocketio.NewServer(*websocket.GetDefaultWebsocketTransport())

	server.On(shadiaosocketio.OnConnection, func(c *shadiaosocketio.Channel) {
		log.Println("received client", c.Id())

		c.Emit("message", Message{10, "main", "using emit"})

		c.Join("room")
		c.BroadcastTo("room", "/admin", Message{10, "main", "using broadcast"})

		server.BroadcastTo("room", "/admin", Message{1, "boss", "hello everyone!"})
	})
	server.On(shadiaosocketio.OnDisconnection, func(c *shadiaosocketio.Channel, reason websocket.CloseError) {
		log.Println("received disconnect", c.Id(), "code:", reason.Code, "text:", reason.Text)
	})

	server.On("message", func(c *shadiaosocketio.Channel, arg1 string, arg2 Message, arg3 int, arg4 bool) {
		log.Println("received arg1:", arg1, "arg2:", arg2, "arg3:", arg3, "arg4:", arg4)
	})

	server.On("/admin", func(c *shadiaosocketio.Channel, channel Channel) (int, string) {
		log.Println("client joined to", channel.Channel, "id:", c.Id())
		return 1, c.Id() + " join success!"
	})

	serveMux := http.NewServeMux()
	serveMux.Handle("/socket.io/", server)

	log.Println("Starting server...")
	log.Panic(http.ListenAndServe(":2233", serveMux))
}
