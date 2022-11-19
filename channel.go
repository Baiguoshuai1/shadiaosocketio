package shadiaosocketio

import (
	"encoding/json"
	"errors"
	"github.com/Baiguoshuai1/shadiaosocketio/protocol"
	"github.com/Baiguoshuai1/shadiaosocketio/websocket"
	"net/http"
	"sync"
	"time"
)

const (
	queueBufferSize = 10000
)
const (
	ManualCloseTxt  = "manual close"
	ManualCloseCode = 101
)

var (
	ErrorWrongHeader = errors.New("Wrong header")
)

/**
engine.io header to send or receive
*/
type Header struct {
	GeSid        string   `json:"geSid"`
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

	out    chan string
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
	c.out = make(chan string, queueBufferSize)
	//c.ack.resultWaiters = make(map[int](chan string))
	c.setAliveValue(true)
}

/**
Get id of current socket connection
*/
func (c *Channel) Sid() string {
	return c.header.GeSid
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

	var s []interface{}
	closeErr := &websocket.CloseError{}

	if len(args) == 0 {
		closeErr.Code = ManualCloseCode
		closeErr.Text = ManualCloseTxt

		s = append(s, closeErr)
	} else {
		s = append(s, args...)
	}

	c.conn.Close()
	c.setAliveValue(false)

	//clean outloop
	for len(c.out) > 0 {
		<-c.out
	}

	c.out <- protocol.Close
	m.callLoopEvent(c, OnDisconnection, s...)

	deleteOverflooded(c)

	return nil
}

//incoming messages loop, puts incoming messages to In channel
func inLoop(c *Channel, m *methods) error {
	for {
		pkg, err := c.conn.GetMessage()
		if err != nil {
			return closeChannel(c, m, err)
		}

		msg, err := protocol.Decode(pkg)
		if err != nil {
			closeErr := &websocket.CloseError{}
			closeErr.Code = websocket.DecodeErrCode
			closeErr.Text = err.Error()

			closeChannel(c, m, closeErr)
			return err
		}

		switch msg.Type {
		case protocol.MessageTypeOpen:
			if err := json.Unmarshal([]byte(msg.Source[1:]), &c.header); err != nil {
				closeErr := &websocket.CloseError{}
				closeErr.Code = websocket.ParseOpenMsgCode
				closeErr.Text = ErrorWrongHeader.Error()

				closeChannel(c, m, closeErr)
			}
			m.callLoopEvent(c, OnConnection)
		case protocol.MessageTypePing:
			c.out <- protocol.Pong
		case protocol.MessageTypePong:
		case protocol.MessageTypeUnKnown:
		default:
			go m.processIncomingMessage(c, msg)
		}
	}
}

var overflooded sync.Map

func deleteOverflooded(c *Channel) {
	overflooded.Delete(c)
}

func storeOverflow(c *Channel) {
	overflooded.Store(c, struct{}{})
}

/**
outgoing messages loop, sends messages from channel to socket
*/
func outLoop(c *Channel, m *methods) error {
	for {
		outBufferLen := len(c.out)
		if outBufferLen >= queueBufferSize-1 {
			closeErr := &websocket.CloseError{}
			closeErr.Code = websocket.QueueBufferSizeCode
			closeErr.Text = ErrorSocketOverflood.Error()

			return closeChannel(c, m, closeErr)
		} else if outBufferLen > int(queueBufferSize/2) {
			storeOverflow(c)
		} else {
			deleteOverflooded(c)
		}

		msg := <-c.out
		if msg == protocol.Close {
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
func pinger(c *Channel) {
	interval, _ := c.conn.PingParams()
	ticker := time.NewTicker(interval)
	for {
		<-ticker.C
		if !c.IsAlive() {
			return
		}
		c.out <- protocol.Ping
	}
}
