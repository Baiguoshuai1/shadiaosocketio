package shadiaosocketio

import (
	"errors"
	"github.com/Baiguoshuai1/shadiaosocketio/protocol"
	"log"
	"time"
)

var (
	ErrorSendTimeout     = errors.New("timeout")
	ErrorSocketOverflood = errors.New("socket overflood")
)

/**
Send message packet to socket
*/
func send(c *Channel, msg *protocol.Message, args ...interface{}) error {
	//preventing json/encoding "index out of range" panic
	defer func() {
		if r := recover(); r != nil {
			log.Println("socket.io send panic: ", r)
		}
	}()

	command, err := protocol.Encode(msg, args...)
	if err != nil {
		return err
	}

	if len(c.out) == queueBufferSize {
		return ErrorSocketOverflood
	}

	c.out <- command

	return nil
}

/**
Create packet based on given data and send it
*/
func (c *Channel) Emit(method string, args ...interface{}) error {
	msg := &protocol.Message{
		Type:   protocol.MessageTypeEmit,
		Method: method,
	}

	return send(c, msg, args...)
}

/**
Create ack packet based on given data and send it and receive response
*/
func (c *Channel) Ack(method string, timeout time.Duration, args ...interface{}) (interface{}, error) {
	msg := &protocol.Message{
		Type:   protocol.MessageTypeAckRequest,
		AckId:  c.ack.getNextId(),
		Method: method,
	}

	waiter := make(chan interface{})
	c.ack.addWaiter(msg.AckId, waiter)

	err := send(c, msg, args...)
	if err != nil {
		c.ack.removeWaiter(msg.AckId)
	}

	select {
	case result := <-waiter:
		return result, nil
	case <-time.After(timeout):
		c.ack.removeWaiter(msg.AckId)
		return "", ErrorSendTimeout
	}
}
