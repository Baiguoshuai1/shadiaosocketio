package shadiaosocketio

import (
	"encoding/json"
	"github.com/Baiguoshuai1/shadiaosocketio/protocol"
	"github.com/buger/jsonparser"
	"reflect"
	"strconv"
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

	f.callFunc(c, 0, args...)
}

func (m *methods) getEventArgs(msg string) (string, []interface{}, error) {
	rawArr := make([]interface{}, 0, 1)
	event := ""
	c := 0

	_, err := jsonparser.ArrayEach([]byte(msg[1:]), func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		c++
		if c == 1 {
			event = string(value)
		}

		if dataType == jsonparser.String {
			// string must add "\"\"" ps: "message"
			marshal, err := json.Marshal("")
			if err != nil {
				return
			}
			v := make([]byte, 0, len(value)+2)
			v = append(v, marshal[0])
			v = append(v, value...)
			v = append(v, marshal[1])

			value = v
		}

		rawArr = append(rawArr, value)
	})

	if err != nil {
		return "", nil, err
	}

	return event, rawArr, nil
}
func (m *methods) processIncomingMessageText(c *Channel, msg string) {
	mType, err := strconv.Atoi(string(msg[0]))
	if err != nil {
		return
	}

	switch mType {
	case protocol.CONNECT:
		sid, err := jsonparser.GetString([]byte(msg[1:]), "sid")
		if err != nil {
			return
		}

		c.header.Sid = sid
		m.callLoopEvent(c, OnConnection)
	case protocol.DISCONNECT:
		closeChannel(c, m)
	case protocol.EVENT:
		event, args, err := m.getEventArgs(msg)
		if err != nil {
			return
		}

		f, ok := m.findMethod(event)
		if !ok {
			return
		}

		if len(args) == 0 {
			f.callFunc(c, 1)
			return
		}

		f.callFunc(c, 1, args[1:]...)

	case protocol.ACK:
		//event, args, err := m.getEventArgs(msg)
		//if err != nil {
		//	log.Println(err)
		//	return
		//}

		//
		//waiter, err := c.ack.getWaiter(packet.Id)
		//
		//if err != nil {
		//	f, ok := m.findMethod(event)
		//	if !ok {
		//		return
		//	}
		//
		//	arr := make([]reflect.Value, 1, 1)
		//	ackRes := f.callFunc(c, 0, data[1:]...)
		//	arr[0] = ackRes[0]
		//
		//	r := &protocol.Message{
		//		Type:   packet.Type,
		//		Method: event,
		//		Nsp:    packet.Nsp,
		//		AckId:  packet.Id,
		//		Args:   []interface{}{arr[0].Interface()},
		//	}
		//
		//	c.out <- protocol.GetMsgPacket(r)
		//} else {
		//	// requester
		//	waiter <- data[1]
		//}
	case protocol.CONNECT_ERROR:
		closeChannel(c, m)
	case protocol.BINARY_EVENT:
	case protocol.BINARY_ACK:
	}
}

func (m *methods) processIncomingMessage(c *Channel, msg string) {
	if !c.conn.GetUseBinaryMessage() {
		go m.processIncomingMessageText(c, msg)
		return
	}

	packet := &protocol.MsgPack{}
	err := json.Unmarshal([]byte(msg), &packet)
	if err != nil {
		return
	}

	switch packet.Type {
	case protocol.CONNECT:
		// server protocol 4 & binary msg -> client protocol 3 // 4{"type":0,"data":null,"nsp":"/","id":0}
		if packet.Data == nil {
			return
		}
		// {"type":0,"data":{},"nsp":"/","id":0}
		if reflect.ValueOf(packet.Data).Len() == 0 {
			return
		}

		c.header.Sid = reflect.ValueOf(packet.Data).MapIndex(reflect.ValueOf("sid")).Interface().(string)
		m.callLoopEvent(c, OnConnection)
	case protocol.DISCONNECT:
		closeChannel(c, m)
	case protocol.EVENT:
		data := packet.Data.([]interface{})
		if len(data) == 0 {
			return
		}
		event := data[0].(string)

		f, ok := m.findMethod(event)
		if !ok {
			return
		}

		if len(data) == 1 {
			f.callFunc(c, 0)
			return
		}

		f.callFunc(c, 0, data[1:]...)

	case protocol.ACK:
		data := packet.Data.([]interface{})
		if len(data) == 0 {
			return
		}
		event := data[0].(string)

		waiter, err := c.ack.getWaiter(packet.Id)

		if err != nil {
			f, ok := m.findMethod(event)
			if !ok {
				return
			}

			arr := make([]reflect.Value, 1, 1)
			ackRes := f.callFunc(c, 0, data[1:]...)
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
		closeChannel(c, m)
	case protocol.BINARY_EVENT:
	case protocol.BINARY_ACK:
	}
}
