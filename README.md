golang socket.io
================

GoLang implementation of [socket.io](http://socket.io) library, client and server.

Examples directory contains simple client and server.

### Get It

```sh
go get -u github.com/Baiguoshuai1/shadiaosocketio
```

### Simple server usage

```go
    server := shadiaosocketio.NewServer(*websocket.GetDefaultWebsocketTransport())
    
    server.On(shadiaosocketio.OnConnection, func(c *shadiaosocketio.Channel) {
        log.Println("received client", c.Id())
    })
	
    server.On(shadiaosocketio.OnDisconnection, func(c *shadiaosocketio.Channel) {
        log.Println("received disconnect", c.Id())
    })
    
    server.On("message", func(c *shadiaosocketio.Channel, arg1 string, arg2 Message, arg3 string) {
        if arg3 == "" {
            log.Println("received \"\" string")
        }
        log.Println("received arg1:", arg1, "arg2.text:", arg2.Text, "arg3:", arg3)
    })
    
    server.On("/admin", func(c *shadiaosocketio.Channel, channel Channel) int {
        log.Println("client joined to", channel.Channel, "id:", c.Id())
        //return c.Id() + " join success!"
        return 666
    })
    
    serveMux := http.NewServeMux()
    serveMux.Handle("/socket.io/", server)
    
    log.Println("Starting server...")
    log.Panic(http.ListenAndServe(":2233", serveMux))
```

### Client

```go
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
		
        time.Sleep(1 * time.Second)
    
        err := c.Emit("message", 1, "arg2")
        if err != nil {
            panic(err)
        }
    })

```

### Javascript client for caller server

```javascript
var socket = io('ws://127.0.0.1:2233', {transports: ['conn']});

    // listen for messages
    socket.on('message', function(message) {

        console.log('new message');
        console.log(message);
    });

    socket.on('connect', function () {

        console.log('socket connected');

        //send something
        socket.emit('send', {name: "my name", message: "hello"}, function(result) {

            console.log('sended successfully');
            console.log(result);
        });
    });
```

## License

MIT
