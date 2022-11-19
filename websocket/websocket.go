package websocket

import (
	"crypto/tls"
	"errors"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
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

func (wsc *Connection) GetMessage() (message string, err error) {
	wsc.socket.SetReadDeadline(time.Now().Add(wsc.transport.ReceiveTimeout))
	msgType, reader, err := wsc.socket.NextReader()
	if err != nil {
		return "", err
	}

	var data []byte

	if msgType == websocket.BinaryMessage {
		data, err = ioutil.ReadAll(reader)

		return "", &websocket.CloseError{
			Code: BinaryMsgErrCode,
			Text: ErrorBinaryMessage.Error(),
		}
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

func (wsc *Connection) WriteMessage(message string) error {
	wsc.socket.SetWriteDeadline(time.Now().Add(wsc.transport.SendTimeout))
	writer, err := wsc.socket.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}

	if _, err := writer.Write([]byte(message)); err != nil {
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

	BufferSize  int
	UnsecureTLS bool

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
		UnsecureTLS:    false,
	}
}
