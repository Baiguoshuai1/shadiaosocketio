package shadiaosocketio

import (
	"encoding/json"
	"github.com/Baiguoshuai1/shadiaosocketio/protocol"
	"github.com/buger/jsonparser"
	"log"
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

func (m *methods) initMethods() {
	//m.messageHandlers = make(sync.Map)
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

	var signal []string
	for _, v := range args {
		marshal, err := json.Marshal(v)
		if err != nil {
			return
		}

		signal = append(signal, string(marshal))
	}

	f.callFunc(c, signal...)
}

func (m *methods) processIncomingMessage(c *Channel, msg *protocol.Message) {
	switch msg.Type {
	case protocol.MessageTypeEmit:
		f, ok := m.findMethod(msg.Method)
		if !ok {
			return
		}

		if msg.Args == "" {
			f.callFunc(c)
			return
		}

		var rawArr []string
		_, err := jsonparser.ArrayEach([]byte(msg.Args), func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			if dataType == jsonparser.String {
				rawArr = append(rawArr, "\""+string(value)+"\"")
			} else {
				rawArr = append(rawArr, string(value))
			}
		})
		if err != nil {
			log.Println("jsonparser error:", err)
			return
		}

		log.Println("[processIncomingMessage]", rawArr)
		f.callFunc(c, rawArr...)

	case protocol.MessageTypeAckRequest:
		f, ok := m.findMethod(msg.Method)

		if !ok {
			return
		}

		var rawArr []string
		_, err := jsonparser.ArrayEach([]byte(msg.Args), func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			if dataType == jsonparser.String {
				rawArr = append(rawArr, "\""+string(value)+"\"")
			} else {
				rawArr = append(rawArr, string(value))
			}
		})
		if err != nil {
			log.Println("jsonparser error:", err)
			return
		}

		ackRes := f.callFunc(c, rawArr...)

		ack := &protocol.Message{
			Type:  protocol.MessageTypeAckResponse,
			AckId: msg.AckId,
		}

		var arr []interface{}
		for i := 0; i < f.NumOut; i++ {
			kind := f.Func.Type().Out(i).Kind()

			if kind == reflect.String {
				arr = append(arr, ackRes[i].String())
			} else if kind == reflect.Int {
				arr = append(arr, ackRes[i].Int())
			} else {
				return
			}
		}

		err = send(c, ack, arr...)
		if err != nil {
			return
		}

	case protocol.MessageTypeAckResponse:
		waiter, err := c.ack.getWaiter(msg.AckId)
		if err == nil {
			var str string
			err := json.Unmarshal([]byte(msg.Args), &str)
			if err != nil {
				log.Println(err)
				return
			}
			waiter <- str
		}
	}
}
