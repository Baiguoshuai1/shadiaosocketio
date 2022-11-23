package shadiaosocketio

import (
	"encoding/json"
	"errors"
	"github.com/Baiguoshuai1/shadiaosocketio/protocol"
	"github.com/Baiguoshuai1/shadiaosocketio/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	queueBufferSize = 10000
)
const (
	DefaultCloseTxt  = "transport close"
	DefaultCloseCode = 101
)

var (
	ErrorWrongHeader = errors.New("Wrong header")
)

/**
engine.io header to send or receive
*/
type Header struct {
	Sid          string   `json:"sid"`
	Upgrades     []string `json:"upgrades"`
	PingInterval int      `json:"pingInterval"`
	PingTimeout  int      `json:"pingTimeout"`
}

/**
socket.io connection handler

use IsAlive to check that handler is still working
use Dial to connect to websocket
use In and Out channels for message exchange
Close message means channel is closed
ping is automatic
*/
type Channel struct {
	conn *websocket.Connection

	out    chan interface{}
	header Header

	alive     bool
	aliveLock sync.Mutex

	ack ackProcessor

	server  *Server
	ip      string
	request *http.Request
}

/**
create channel, map, and set active
*/
func (c *Channel) initChannel() {
	//TODO: queueBufferSize from constant to server or socket variable
	c.out = make(chan interface{}, queueBufferSize)
	//c.ack.resultWaiters = make(map[int](chan string))
	c.setAliveValue(true)
}

func (c *Channel) Id() string {
	return c.header.Sid
}

/**
Checks that Channel is still alive
*/
func (c *Channel) IsAlive() bool {
	c.aliveLock.Lock()
	isAlive := c.alive
	c.aliveLock.Unlock()

	return isAlive
}

func (c *Channel) setAliveValue(value bool) {
	c.aliveLock.Lock()
	c.alive = value
	c.aliveLock.Unlock()
}

/**
Close channel
*/
func closeChannel(c *Channel, m *methods, args ...interface{}) error {
	if !c.IsAlive() {
		//already closed
		return nil
	}
	c.setAliveValue(false)

	var s []interface{}
	closeErr := &websocket.CloseError{}

	if len(args) == 0 {
		closeErr.Code = DefaultCloseCode
		closeErr.Text = DefaultCloseTxt

		s = append(s, closeErr)
	} else {
		s = append(s, args...)
	}

	c.conn.Close()

	//clean outloop
	for len(c.out) > 0 {
		<-c.out
	}

	m.callLoopEvent(c, OnDisconnection, s...)

	return nil
}

//incoming messages loop, puts incoming messages to In channel
func inLoop(c *Channel, m *methods) error {
	for {
		msg, err := c.conn.GetMessage()
		if err != nil {
			return closeChannel(c, m, err)
		}
		prefix := string(msg[0])
		protocolV := c.conn.GetProtocol()

		log.Println("[inLoop]", msg)
		switch prefix {
		case protocol.OpenMsg:
			if err := json.Unmarshal([]byte(msg[1:]), &c.header); err != nil {
				closeErr := &websocket.CloseError{}
				closeErr.Code = websocket.ParseOpenMsgCode
				closeErr.Text = err.Error()

				return closeChannel(c, m, closeErr)
			}

			if protocolV == protocol.Protocol3 {
				m.callLoopEvent(c, OnConnection)
				// in protocol v3, the client sends a ping, and the server answers with a pong
				go schedulePing(c)
			}
			if c.conn.GetProtocol() == protocol.Protocol4 {
				// in protocol v4 & binary msg Connection to a namespace
				if c.conn.GetUseBinaryMessage() {
					c.out <- &protocol.MsgPack{
						Type: protocol.CONNECT,
						Nsp:  "/",
						Data: &struct {
						}{},
					}
					// in protocol v4 & text msg Connection to a namespace
				} else {
					c.out <- protocol.CommonMsg + protocol.OpenMsg
				}
			}
		case protocol.CloseMsg:
			return closeChannel(c, m)
		case protocol.PingMsg:
			// in protocol v4, the server sends a ping, and the client answers with a pong
			c.out <- protocol.PongMsg
		case protocol.PongMsg:
		case protocol.UpgradeMsg:
		case protocol.CommonMsg:
			// in protocol v3 & binary msg  ps: 4{"type":0,"data":null,"nsp":"/","id":0}
			// in protocol v3 & text msg  ps: 40 or 41 or 42["message", ...]
			// in protocol v4 & text msg  ps: 40 or 41 or 42["message", ...]
			go m.processIncomingMessage(c, msg[1:])
		default:
			// in protocol v4 & binary msg ps: {"type":0,"data":{"sid":"HWEr440000:1:R1CHyink:shadiao:101"},"nsp":"/","id":0}
			go m.processIncomingMessage(c, msg)
		}
	}
}

func outLoop(c *Channel, m *methods) error {
	for {
		outBufferLen := len(c.out)
		if outBufferLen >= queueBufferSize-1 {
			closeErr := &websocket.CloseError{}
			closeErr.Code = websocket.QueueBufferSizeCode
			closeErr.Text = ErrorSocketOverflood.Error()

			return closeChannel(c, m, closeErr)
		}

		msg := <-c.out
		if msg == protocol.CloseMsg {
			return nil
		}

		err := c.conn.WriteMessage(msg)
		if err != nil {
			closeErr := &websocket.CloseError{}
			closeErr.Code = websocket.WriteBufferErrCode
			closeErr.Text = err.Error()

			return closeChannel(c, m, closeErr)
		}
	}
}

/**
Pinger sends ping messages for keeping connection alive
*/
func schedulePing(c *Channel) {
	interval, _ := c.conn.PingParams()
	ticker := time.NewTicker(interval)
	for {
		<-ticker.C
		if !c.IsAlive() {
			return
		}
		c.out <- protocol.PingMsg
	}
}
