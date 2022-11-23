package protocol

// https://github.com/socketio/socket.io-protocol#connection-to-a-namespace
const (
	Protocol3 = 3
	Protocol4 = 4
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
