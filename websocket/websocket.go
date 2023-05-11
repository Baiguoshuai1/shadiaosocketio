package websocket

import (
	"bytes"
	"crypto/tls"
	"errors"
	"github.com/Baiguoshuai1/shadiaosocketio/protocol"
	"github.com/Baiguoshuai1/shadiaosocketio/utils"
	"github.com/gorilla/websocket"
	"github.com/ugorji/go/codec"
	"io"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"time"
)

const (
	upgradeFailed = "Upgrade failed: "

	WsDefaultPingInterval   = 30 * time.Second
	WsDefaultPingTimeout    = 60 * time.Second
	WsDefaultReceiveTimeout = 60 * time.Second
	WsDefaultSendTimeout    = 60 * time.Second
	WsDefaultBufferSize     = 1024 * 32

	maxRecordReadBytes  = 1024 * 1024 * 1024
	maxRecordWriteBytes = 1024 * 1024 * 1024
)

const (
	DecodeErrCode       = 102
	ParseOpenMsgCode    = 103
	QueueBufferSizeCode = 104
	WriteBufferErrCode  = 105
	BinaryMsgErrCode    = 106
	BadBufferErrCode    = 107
	PacketWrongErrCode  = 108
)

var (
	ErrorBinaryMessage     = errors.New("binary messages are not supported")
	ErrorBadBuffer         = errors.New("buffer error")
	ErrorPacketWrong       = errors.New("wrong packet type error")
	ErrorMethodNotAllowed  = errors.New("method not allowed")
	ErrorHttpUpgradeFailed = errors.New("http upgrade failed")
)

// create and configure Handle
var (
	mh codec.MsgpackHandle
)

type CloseError struct {
	websocket.CloseError
}

type Connection struct {
	socket     *websocket.Conn
	transport  *Transport
	writeBytes int
	readBytes  int
}

func (wsc *Connection) RemoteAddr() net.Addr {
	return wsc.socket.RemoteAddr()
}

func (wsc *Connection) LocalAddr() net.Addr {
	return wsc.socket.LocalAddr()
}

func (wsc *Connection) GetProtocol() int {
	return wsc.transport.Protocol
}

func (wsc *Connection) GetUseBinaryMessage() bool {
	return wsc.transport.BinaryMessage
}

func (wsc *Connection) GetReadBytes() int {
	v := wsc.readBytes
	wsc.readBytes = 0
	return v
}

func (wsc *Connection) GetWriteBytes() int {
	v := wsc.writeBytes
	wsc.writeBytes = 0
	return v
}

func (wsc *Connection) GetMessage() (message string, err error) {
	err = wsc.socket.SetReadDeadline(time.Now().Add(wsc.transport.ReceiveTimeout))
	if err != nil {
		return "", err
	}

	msgType, reader, err := wsc.socket.NextReader()
	if err != nil {
		return "", err
	}

	var data []byte
	data, err = io.ReadAll(reader)
	if err != nil {
		return "", &websocket.CloseError{
			Code: BadBufferErrCode,
			Text: err.Error(),
		}
	}

	utils.Debug("[GetMessage]", data)
	if wsc.readBytes > maxRecordReadBytes {
		wsc.readBytes = 0
	}
	wsc.readBytes += len(data)
	return wsc.decodeMessage(data, msgType)
}

func (wsc *Connection) WriteMessage(message interface{}) error {
	utils.Debug("[WriteMessage]", message)

	err := wsc.socket.SetWriteDeadline(time.Now().Add(wsc.transport.SendTimeout))
	if err != nil {
		return err
	}

	var data []byte

	messageType := websocket.TextMessage
	if reflect.TypeOf(message).Kind() == reflect.String {
		data = []byte(message.(string))
	} else {
		if wsc.transport.BinaryMessage {
			messageType = websocket.BinaryMessage
		} else {
			messageType = websocket.TextMessage
		}

		data, err = wsc.encodeMessage(message.(*protocol.MsgPack), messageType)
		if err != nil {
			return err
		}
	}

	writer, err := wsc.socket.NextWriter(messageType)
	if err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	if wsc.writeBytes > maxRecordWriteBytes {
		wsc.writeBytes = 0
	}
	wsc.writeBytes += len(data)
	return nil
}

func (wsc *Connection) decodeMessage(data []byte, messageType int) (string, error) {
	if messageType == websocket.TextMessage {
		utils.Debug("[decodeMessage]", string(data))
		return string(data), nil
	}
	prefix := ""

	if wsc.transport.Protocol == protocol.Protocol3 {
		prefix = strconv.Itoa(int(data[0]))
		data = data[1:]
	}

	var m = protocol.MsgPack{
		Id: -1,
	}
	dec := codec.NewDecoderBytes(data, &mh)
	err := dec.Decode(&m)
	if err != nil {
		return "", &websocket.CloseError{
			Code: DecodeErrCode,
			Text: err.Error(),
		}
	}

	msg, err := utils.Json.MarshalToString(&m)
	if err != nil {
		return "", &websocket.CloseError{
			Code: DecodeErrCode,
			Text: err.Error(),
		}
	}

	utils.Debug("[decodeMessage]", prefix+msg)
	return prefix + msg, nil
}

func (wsc *Connection) encodeMessage(msg *protocol.MsgPack, messageType int) ([]byte, error) {
	// {
	//  "type": 3,
	//  "nsp": "/admin",
	//  "data": [],
	//  "id": 456
	// }
	// is encoded to 3/admin,456[]

	packet := ""
	if messageType == websocket.TextMessage {
		// Engine.IO Flag
		prefix := protocol.CommonMsg
		// Socket.IO Flag
		event := strconv.Itoa(msg.Type)
		ackId := strconv.Itoa(msg.Id)
		data, _ := utils.Json.Marshal(&msg.Data)

		// sending ack res or sending ack req
		if msg.Type == protocol.ACK || msg.Id >= 0 {
			packet = prefix + event + ackId + string(data)
		} else {
			packet = prefix + event + string(data)
		}

		utils.Debug("[encodeMessage]", packet)
		return []byte(packet), nil
	}

	buf := bytes.Buffer{}
	enc := codec.NewEncoder(&buf, &mh)
	err := enc.Encode(msg)
	if err != nil {
		return nil, err
	}

	prefix, err := strconv.ParseUint(protocol.CommonMsg, 10, 8)
	if err != nil {
		return nil, err
	}

	bf := make([]uint8, 0, buf.Len())
	bf = append(bf, uint8(prefix))
	bf = append(bf, buf.Bytes()...)

	if wsc.transport.Protocol == protocol.Protocol4 {
		bf = bf[1:]
	}

	return bf, nil
}

func (wsc *Connection) Close() {
	wsc.socket.Close()
}

func (wsc *Connection) PingParams() (interval, timeout time.Duration) {
	return wsc.transport.PingInterval, wsc.transport.PingTimeout
}

type Transport struct {
	PingInterval   time.Duration
	PingTimeout    time.Duration
	ReceiveTimeout time.Duration
	SendTimeout    time.Duration

	Protocol      int
	BufferSize    int
	BinaryMessage bool

	UnsecureTLS   bool
	TLSConfig     *tls.Config

	RequestHeader http.Header
}

func (wst *Transport) Connect(url string) (conn *Connection, err error) {
	tlsCfg := wst.TLSConfig
	if tlsCfg == nil {
		tlsCfg = &tls.Config{InsecureSkipVerify: wst.UnsecureTLS}
	}
	dialer := websocket.Dialer{TLSClientConfig: tlsCfg}
	socket, _, err := dialer.Dial(url, wst.RequestHeader)
	if err != nil {
		return nil, err
	}

	return &Connection{socket, wst, 0, 0}, nil
}

func (wst *Transport) HandleConnection(
	w http.ResponseWriter, r *http.Request) (conn *Connection, err error) {

	if r.Method != "GET" {
		http.Error(w, upgradeFailed+ErrorMethodNotAllowed.Error(), 503)
		return nil, ErrorMethodNotAllowed
	}

	upgrade := &websocket.Upgrader{
		ReadBufferSize:  wst.BufferSize,
		WriteBufferSize: wst.BufferSize,
	}
	socket, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, upgradeFailed+err.Error(), 503)
		return nil, ErrorHttpUpgradeFailed
	}

	return &Connection{socket, wst, 0, 0}, nil
}

/*
*
Websocket connection do not require any additional processing
*/
func (wst *Transport) Serve(w http.ResponseWriter, r *http.Request) {}

/*
*
Returns websocket connection with default params
*/
func GetDefaultWebsocketTransport() *Transport {
	return &Transport{
		Protocol:       protocol.Protocol4,
		PingInterval:   WsDefaultPingInterval,
		PingTimeout:    WsDefaultPingTimeout,
		ReceiveTimeout: WsDefaultReceiveTimeout,
		SendTimeout:    WsDefaultSendTimeout,
		BufferSize:     WsDefaultBufferSize,
		BinaryMessage:  false,
		UnsecureTLS:    false,
		TLSConfig:      nil,
	}
}
