package protocol

// https://github.com/socketio/socket.io-protocol#connection-to-a-namespace
const (
	Protocol3 = 3
	Protocol4 = 4
)

func GetMsgPacket(msg *Message) *MsgPack {
	data := make([]interface{}, 1, 1+len(msg.Args))
	if msg.Method != "" {
		data[0] = msg.Method
	} else {
		// sending ack res
		data = data[1:]
	}
	data = append(data, msg.Args...)

	return &MsgPack{
		Type: msg.Type,
		Data: data,
		Nsp:  msg.Nsp,
		Id:   msg.AckId,
	}
}
