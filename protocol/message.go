package protocol

// https://github.com/socketio/socket.io-protocol#difference-between-v3-and-v2

// msg type
const (
	CONNECT       = 0
	DISCONNECT    = 1
	EVENT         = 2
	ACK           = 3
	CONNECT_ERROR = 4
	BINARY_EVENT  = 5
	BINARY_ACK    = 6
)

const DefaultNsp = "/"

// msg value
const (
	OpenMsg    = "0"
	CloseMsg   = "1"
	PingMsg    = "2"
	PongMsg    = "3"
	CommonMsg  = "4"
	UpgradeMsg = "5"
)

type MsgPack struct {
	Type int         `json:"type"`
	Data interface{} `json:"data"`
	Nsp  string      `json:"nsp"`
	Id   int         `json:"id"`
}

type Message struct {
	Type   int
	AckId  int
	Method string
	Nsp    string
	Args   []interface{}
}
