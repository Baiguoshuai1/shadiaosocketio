package websocket

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	upgradeFailed = "Upgrade failed: "

	WsDefaultPingInterval   = 30 * time.Second
	WsDefaultPingTimeout    = 60 * time.Second
	WsDefaultReceiveTimeout = 60 * time.Second
	WsDefaultSendTimeout    = 60 * time.Second
	WsDefaultBufferSize     = 1024 * 32
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

type CloseError struct {
	websocket.CloseError
}

type Connection struct {
	socket    *websocket.Conn
	transport *Transport
}

type MsgPack struct {
	Type int         `json:"type"`
	Data interface{} `json:"data"`
	Nsp  string      `json:"nsp"`
}

func (wsc *Connection) GetMessage() (message string, err error) {
	wsc.socket.SetReadDeadline(time.Now().Add(wsc.transport.ReceiveTimeout))
	msgType, reader, err := wsc.socket.NextReader()
	if err != nil {
		return "", err
	}

	var data []byte

	if msgType == websocket.BinaryMessage {
		data, err = ioutil.ReadAll(reader)
		if err != nil {
			return "", err
		}

		str, err := decodeBinaryMessage(data)
		return str, err
	} else {
		data, err = ioutil.ReadAll(reader)
		if err != nil {
			return "", &websocket.CloseError{
				Code: BadBufferErrCode,
				Text: ErrorBadBuffer.Error(),
			}
		}
	}

	text := string(data)

	if len(text) == 0 {
		return "", &websocket.CloseError{
			Code: PacketWrongErrCode,
			Text: ErrorPacketWrong.Error(),
		}
	}

	return text, nil
}

func decodeBinaryMessage(data []byte) (string, error) {
	prefix := strconv.Itoa(int(data[0]))

	buf := bytes.NewBuffer(data[1:])
	dec := msgpack.NewDecoder(buf)
	dec.SetCustomStructTag("json")

	var m MsgPack
	err := dec.Decode(&m)
	if err != nil {
		return "", err
	}

	mType := strconv.Itoa(m.Type)
	marshal, err := json.Marshal(m.Data)
	if err != nil {
		return "", err
	}

	return prefix + mType + string(marshal), nil
}

func encodeBinaryMessage(msg string) ([]byte, error) {
	buf := bytes.Buffer{}
	enc := msgpack.NewEncoder(&buf)
	enc.SetCustomStructTag("json")

	pos := strings.IndexByte(msg, '[')
	data := make([]interface{}, 0, 2)

	_, err := jsonparser.ArrayEach([]byte(msg[pos:]), func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		switch dataType {
		case jsonparser.Number:
			parseInt, _ := jsonparser.ParseInt(value)
			data = append(data, parseInt)
		case jsonparser.Boolean:
			bo, _ := jsonparser.ParseBoolean(value)
			data = append(data, bo)
		default:
			data = append(data, string(value))
		}
	})

	prefix, err := strconv.ParseUint(string(msg[0]), 10, 8)
	if err != nil {
		return nil, err
	}
	mType, err := strconv.Atoi(string(msg[1]))
	if err != nil {
		return nil, err
	}

	err = enc.Encode(&MsgPack{
		Type: mType,
		Data: data,
		Nsp:  "/",
	})
	if err != nil {
		return []byte{}, err
	}

	bf := make([]uint8, 0, 1+buf.Len())

	bf = append(bf, uint8(prefix))
	bf = append(bf, buf.Bytes()...)

	return bf, nil
}

func (wsc *Connection) WriteMessage(message string) error {
	wsc.socket.SetWriteDeadline(time.Now().Add(wsc.transport.SendTimeout))

	var err error
	var data []byte

	messageType := websocket.TextMessage

	if len(message) <= 2 || message[0] == '0' {
		messageType = websocket.TextMessage
		data = []byte(message)
	} else {
		if wsc.transport.BinaryMessage {
			messageType = websocket.BinaryMessage
			data, err = encodeBinaryMessage(message)
			if err != nil {
				return err
			}
		} else {
			data = []byte(message)
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
	return nil
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

	BufferSize    int
	UnsecureTLS   bool
	BinaryMessage bool

	RequestHeader http.Header
}

func (wst *Transport) Connect(url string) (conn *Connection, err error) {
	dialer := websocket.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: wst.UnsecureTLS}}
	socket, _, err := dialer.Dial(url, wst.RequestHeader)
	if err != nil {
		return nil, err
	}

	return &Connection{socket, wst}, nil
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

	return &Connection{socket, wst}, nil
}

/**
Websocket connection do not require any additional processing
*/
func (wst *Transport) Serve(w http.ResponseWriter, r *http.Request) {}

/**
Returns websocket connection with default params
*/
func GetDefaultWebsocketTransport() *Transport {
	return &Transport{
		PingInterval:   WsDefaultPingInterval,
		PingTimeout:    WsDefaultPingTimeout,
		ReceiveTimeout: WsDefaultReceiveTimeout,
		SendTimeout:    WsDefaultSendTimeout,
		BufferSize:     WsDefaultBufferSize,
		BinaryMessage:  false,
		UnsecureTLS:    false,
	}
}
