package shadiaosocketio

import (
	"github.com/Baiguoshuai1/shadiaosocketio/websocket"
	"net"
	"strconv"
)

const (
	webSocketProtocol       = "ws://"
	webSocketSecureProtocol = "wss://"
	socketIOUrl             = "/socket.io/?EIO=3&transport=websocket"
)

type Client struct {
	methods
	Channel
}

func GetUrl(host string, port int, secure bool) string {
	var prefix string
	if secure {
		prefix = webSocketSecureProtocol
	} else {
		prefix = webSocketProtocol
	}
	return prefix + net.JoinHostPort(host, strconv.Itoa(port)) + socketIOUrl
}

func Dial(url string, tr websocket.Transport) (*Client, error) {
	c := &Client{}
	c.initChannel()

	var err error
	c.conn, err = tr.Connect(url)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	go inLoop(&c.Channel, &c.methods)
	go outLoop(&c.Channel, &c.methods)
	go pinger(&c.Channel)

	return c, nil
}

func (c *Client) GenerateNewId(id string) {
	c.header.NewSid = id
}

func (c *Client) Close() {
	closeChannel(&c.Channel, &c.methods)
}
