package shadiaosocketio

import (
	"encoding/json"
	"github.com/Baiguoshuai1/shadiaosocketio/protocol"
	"reflect"
	"sync"
)

const (
	OnMessage       = "message"
	OnConnection    = "connection"
	OnDisconnection = "disconnection"
	OnError         = "error"
)

/**
System handler function for internal event processing
*/
type systemHandler func(c *Channel)

/**
Contains maps of message processing functions
*/
type methods struct {
	messageHandlers     sync.Map
	messageHandlersLock sync.RWMutex

	onConnection    systemHandler
	onDisconnection systemHandler
}

func (m *methods) On(method string, f interface{}) error {
	c, err := newCaller(f)
	if err != nil {
		return err
	}

	m.messageHandlers.Store(method, c)
	return nil
}

/**
Find message processing function associated with given method
*/
func (m *methods) findMethod(method string) (*caller, bool) {
	if f, ok := m.messageHandlers.Load(method); ok {
		return f.(*caller), true
	}

	return nil, false
}

func (m *methods) callLoopEvent(c *Channel, event string, args ...interface{}) {
	if m.onConnection != nil && event == OnConnection {
		m.onConnection(c)
	}
	if m.onDisconnection != nil && event == OnDisconnection {
		m.onDisconnection(c)
	}

	f, ok := m.findMethod(event)
	if !ok {
		return
	}

	f.callFunc(c, args...)
}

func (m *methods) processIncomingMessage(c *Channel, msg string) {
	packet := &protocol.MsgPack{}
	err := json.Unmarshal([]byte(msg), &packet)
	if err != nil {
		return
	}
	if packet.Data == nil {
		return
	}

	data := packet.Data.([]interface{})
	if len(data) == 0 {
		return
	}
	event := data[0].(string)

	switch packet.Type {
	case protocol.CONNECT:
	case protocol.DISCONNECT:
	case protocol.EVENT:
		f, ok := m.findMethod(event)
		if !ok {
			return
		}

		if len(data) == 1 {
			f.callFunc(c)
			return
		}

		f.callFunc(c, data[1:]...)

	case protocol.ACK:
		waiter, err := c.ack.getWaiter(packet.Id)

		if err != nil {
			f, ok := m.findMethod(event)
			if !ok {
				return
			}

			arr := make([]reflect.Value, 1, 1)
			ackRes := f.callFunc(c, data[1:]...)
			arr[0] = ackRes[0]

			r := &protocol.Message{
				Type:   packet.Type,
				Method: event,
				Nsp:    packet.Nsp,
				AckId:  packet.Id,
				Args:   []interface{}{arr[0].Interface()},
			}

			c.out <- protocol.GetMsgPacket(r)
		} else {
			// requester
			waiter <- data[1]
		}
	case protocol.CONNECT_ERROR:
	case protocol.BINARY_EVENT:
	case protocol.BINARY_ACK:

	}
}
