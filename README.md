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
		log.Println("[server] connected! id:", c.Id())
		log.Println("[server]", c.LocalAddr().Network()+" "+c.LocalAddr().String()+
			" --> "+c.RemoteAddr().Network()+" "+c.RemoteAddr().String())

		c.Join("room")
		server.BroadcastTo("room", "/admin", Message{1, "new members!"})
		time.Sleep(100 * time.Millisecond)
		c.BroadcastTo("room", "/admin", Message{2, "hello everyone!"})

		_ = c.Emit("message", Message{10, "server channel"})

		// return [][]byte
		result, err := c.Ack("/ackFromServer", time.Second*5, "go", 3)
		if err != nil {
			log.Println("[server] ack cb err:", err)
			return
		}
		log.Println("[server] ack cb:", result)
	})
	server.On(shadiaosocketio.OnDisconnection, func(c *shadiaosocketio.Channel, reason websocket.CloseError) {
		log.Println("[server] received disconnect", c.Id(), "code:", reason.Code, "text:", reason.Text)
	})

	server.On("message", func(c *shadiaosocketio.Channel, arg1 string, arg2 Message, arg3 int, arg4 bool) {
		log.Println("[server] received message:", "arg1:", arg1, "arg2:", arg2, "arg3:", arg3, "arg4:", arg4)
	})

	// listen ack event
	server.On("/ackFromClient", func(c *shadiaosocketio.Channel, msg Message, num int) (int, Desc, string) {
		log.Println("[server] received ack:", msg, num)
		return 1, Desc{Text: "resp"}, "server"
	})

	serveMux := http.NewServeMux()
	serveMux.Handle("/socket.io/", server)

	log.Println("[server] starting server...")
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
		log.Println("[client] got chat message:", args)
	})
	_ = c.On("/admin", func(h *shadiaosocketio.Channel, args Message) {
		log.Println("[client] got admin message:", args)
	})
	// sending ack response
	_ = c.On("/ackFromServer", func(h *shadiaosocketio.Channel, arg1 string, arg2 int) (Message, int) {
		log.Println("[client] got ack from server:", arg1, arg2)
		time.Sleep(3 * time.Second)
		return Message{
			Id:      5,
			Channel: "client",
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

	select {}
}
```

### JavaScript Client For Caller Server

```javascript
const io = require("socket.io-client")
const socket = io("ws://127.0.0.1:2233",{transports: ['websocket']})

// listen for messages
socket.on('message', function(msg) {
    console.log('[client] received msg:', msg);
});

// sending ack response
socket.on('/ackFromServer', function(a, b, f) {
    console.log('[client] received ack:', a , b);
    f({ id: 5, channel: 'js channel' }, 6)
});

socket.on('/admin', function(msg) {
    console.log('[client] received admin msg:', msg);
});

socket.on('connect', function () {
    console.log('[client] socket connected');

    socket.emit('message', "js", { id: 1, channel: "js" }, 2);

    // ack
    socket.emit('/ackFromClient',  { id: 3, channel: "js ack" }, 4, (a, b, c) => {
        console.log('[client] ack cb:', a, b, c)
    });
});

socket.on('disconnect', function (e) {
    console.log('[client] socket disconnect', e);
});

socket.on('connect_error', function (e) {
    console.log('[client] connect_error', e)
});
```

### JavaScript Server
````javascript
const { Server } = require("socket.io");
const customParser = require('socket.io-msgpack-parser');

const io = new Server(2233, {
    // parser: customParser // enable binary msg
});

io.on("connection", (socket) => {
    // ...
    socket.on('message', function(arg1, arg2, arg3) {
       console.log('[server] received message:', arg1, arg2, arg3)
    });

    // listen ack event
    socket.on('/ackFromClient', function(arg1, arg2, func) {
        console.log('[server] received ack:', arg1, arg2)
        func(1, { text: 'resp' }, "server")
    });

    socket.emit('message', { id: 2, channel: 'server channel'});

    // emit ack event
    socket.emit('/ackFromServer', "go", 3, function(arg1, arg2) {
        console.log('[server] ack cb:', arg1, arg2)
    });
});

console.log("[server] starting server...")
````

## License

MIT
