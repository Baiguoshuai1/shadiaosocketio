golang socket.io
================

GoLang implementation of [socket.io](http://socket.io) library, client and server.

Examples directory contains simple client and server.

### Get It

```sh
go get -u github.com/Baiguoshuai1/shadiaosocketio
```

### Debug
```sh
DEBUG=1 go run server.go
```

### Simple server usage

```go
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
```

### Client

```go
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
		log.Println("Connected!", h.Id())
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
```

### Javascript client for caller server

```javascript
const io = require("socket.io-client")
const socket = io("ws://127.0.0.1:2233",{transports: ['websocket']})

// listen for messages
socket.on('message', function(msg) {
    console.log('received msg:', msg);
});

socket.on('/admin', function(msg) {
    console.log('received admin msg:', msg);
});

socket.on('connect', function () {
    console.log('socket connected');

    socket.emit('message', "1", { id: 2, text: "js" }, 3);
});

socket.on('disconnect', function (e) {
    console.log('socket disconnect', e);
});

socket.on('connect_error', function (e) {
    console.log('connect_error', e)
});
```

## License

MIT
