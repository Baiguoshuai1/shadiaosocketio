package protocol

import (
	"errors"
)

// https://github.com/socketio/socket.io-protocol#difference-between-v3-and-v2

var (
	ErrorWrongMessageType = errors.New("wrong message type")
	ErrorWrongPacket      = errors.New("wrong packet")
)

func GetMsgPacket(msg *Message) *MsgPack {
	data := make([]interface{}, 0, 1+len(msg.Args))
	data = append(data, msg.Method)
	data = append(data, msg.Args...)

	return &MsgPack{
		Type: msg.Type,
		Data: data,
		Nsp:  msg.Nsp,
		Id:   msg.AckId,
	}
}
