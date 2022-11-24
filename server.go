package shadiaosocketio

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Baiguoshuai1/shadiaosocketio/protocol"
	"github.com/Baiguoshuai1/shadiaosocketio/websocket"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const (
	HeaderForward = "X-Forwarded-For"
)

var (
	ErrorServerNotSet       = errors.New("server not set")
	ErrorConnectionNotFound = errors.New("connection not found")
)

/**
socket.io server instance
*/
type Server struct {
	methods
	http.Handler

	headers      map[string]string
	channels     map[string]map[*Channel]struct{}
	rooms        map[*Channel]map[string]struct{}
	channelsLock sync.RWMutex

	sids     map[string]*Channel
	sidsLock sync.RWMutex

	tr websocket.Transport
}

/**
Close current channel
*/
func (c *Channel) Close() {
	if c.server != nil {
		closeChannel(c, &c.server.methods)
	}
}

/**
Get ip of socket socket
*/
func (c *Channel) Ip() string {
	forward := c.RequestHeader().Get(HeaderForward)
	if forward != "" {
		return forward
	}
	return c.ip
}

/**
Get request header of this connection
*/
func (c *Channel) RequestHeader() http.Header {
	return c.request.Header
}

/**
Get request
*/
func (c *Channel) Request() *http.Request {
	return c.request
}

/**
Get channel by it's sid
*/
func (s *Server) GetChannel(sid string) (*Channel, error) {
	s.sidsLock.RLock()
	defer s.sidsLock.RUnlock()

	c, ok := s.sids[sid]
	if !ok {
		return nil, ErrorConnectionNotFound
	}

	return c, nil
}

/**
Join this channel to given room
*/
func (c *Channel) Join(room string) error {
	if c.server == nil {
		return ErrorServerNotSet
	}

	c.server.channelsLock.Lock()
	defer c.server.channelsLock.Unlock()

	cn := c.server.channels
	if _, ok := cn[room]; !ok {
		cn[room] = make(map[*Channel]struct{})
	}

	byRoom := c.server.rooms
	if _, ok := byRoom[c]; !ok {
		byRoom[c] = make(map[string]struct{})
	}

	cn[room][c] = struct{}{}
	byRoom[c][room] = struct{}{}

	return nil
}

/**
Remove this channel from given room
*/
func (c *Channel) Leave(room string) error {
	if c.server == nil {
		return ErrorServerNotSet
	}

	c.server.channelsLock.Lock()
	defer c.server.channelsLock.Unlock()

	cn := c.server.channels
	if _, ok := cn[room]; ok {
		delete(cn[room], c)
		if len(cn[room]) == 0 {
			delete(cn, room)
		}
	}

	byRoom := c.server.rooms
	if _, ok := byRoom[c]; ok {
		delete(byRoom[c], room)
	}

	return nil
}

/**
Get amount of channels, joined to given room, using channel
*/
func (c *Channel) Amount(room string) int {
	if c.server == nil {
		return 0
	}

	return c.server.Amount(room)
}

/**
Get amount of channels, joined to given room, using server
*/
func (s *Server) Amount(room string) int {
	s.channelsLock.RLock()
	defer s.channelsLock.RUnlock()

	roomChannels, _ := s.channels[room]
	return len(roomChannels)
}

/**
Get list of channels, joined to given room, using channel
*/
func (c *Channel) List(room string) []*Channel {
	if c.server == nil {
		return []*Channel{}
	}

	return c.server.List(room)
}

/**
Get list of channels, joined to given room, using server
*/
func (s *Server) List(room string) []*Channel {
	s.channelsLock.RLock()
	defer s.channelsLock.RUnlock()

	roomChannels, ok := s.channels[room]
	if !ok {
		return []*Channel{}
	}

	i := 0
	roomChannelsCopy := make([]*Channel, len(roomChannels))
	for channel := range roomChannels {
		roomChannelsCopy[i] = channel
		i++
	}

	return roomChannelsCopy

}

func (c *Channel) BroadcastTo(room, method string, args interface{}) {
	if c.server == nil {
		return
	}

	c.server.channelsLock.RLock()
	defer c.server.channelsLock.RUnlock()

	roomChannels, ok := c.server.channels[room]
	if !ok {
		return
	}

	for cn := range roomChannels {
		if cn.Id() != c.Id() && cn.IsAlive() {
			go cn.Emit(method, args)
		}
	}
}

/**
Broadcast message to all room channels
*/
func (s *Server) BroadcastTo(room, method string, args interface{}) {
	s.channelsLock.RLock()
	defer s.channelsLock.RUnlock()

	roomChannels, ok := s.channels[room]
	if !ok {
		return
	}

	for cn := range roomChannels {
		if cn.IsAlive() {
			go cn.Emit(method, args)
		}
	}
}

/**
Broadcast to all clients
*/
func (s *Server) BroadcastToAll(method string, args interface{}) {
	s.sidsLock.RLock()
	defer s.sidsLock.RUnlock()

	for _, cn := range s.sids {
		if cn.IsAlive() {
			go cn.Emit(method, args)
		}
	}
}

/**
Generate new id for socket.io connection
*/
func generateNewId(custom string) string {
	hash := fmt.Sprintf("%s %s %n %n", custom, time.Now(), rand.Uint32(), rand.Uint32())
	buf := bytes.NewBuffer(nil)
	sum := md5.Sum([]byte(hash))
	encoder := base64.NewEncoder(base64.URLEncoding, buf)
	encoder.Write(sum[:])
	encoder.Close()
	return buf.String()[:20]
}

/**
On connection system handler, store sid
*/
func onConnectStore(c *Channel) {
	c.server.sidsLock.Lock()
	defer c.server.sidsLock.Unlock()

	c.server.sids[c.Id()] = c

	if c.conn.GetProtocol() == protocol.Protocol4 {
		// in protocol v4, the server sends a ping, and the client answers with a pong
		go SchedulePing(c)
	}
}

/**
On disconnection system handler, clean joins and sid
*/
func onDisconnectCleanup(c *Channel) {
	c.server.channelsLock.Lock()
	defer c.server.channelsLock.Unlock()

	cn := c.server.channels
	byRoom, ok := c.server.rooms[c]
	if ok {
		for room := range byRoom {
			if curRoom, ok := cn[room]; ok {
				delete(curRoom, c)
				if len(curRoom) == 0 {
					delete(cn, room)
				}
			}
		}

		delete(c.server.rooms, c)
	}

	go deleteSid(c)
}

func deleteSid(c *Channel) {
	c.server.sidsLock.Lock()
	defer c.server.sidsLock.Unlock()

	delete(c.server.sids, c.Id())
}

func (s *Server) SendOpenSequence(c *Channel) {
	jsonHdr, err := json.Marshal(&c.header)
	if err != nil {
		panic(err)
	}

	// GET /socket.io/?EIO=4&transport=polling&t=N8hyd6w
	// < HTTP/1.1 200 OK
	// < Content-Type: text/plain; charset=UTF-8
	// 0{"sid":"lv_VI97HAXpY6yYWAAAC","upgrades":["websocket"],"pingInterval":25000,"pingTimeout":5000,"maxPayload":1000000}
	c.out <- protocol.OpenMsg + string(jsonHdr)

	if s.tr.BinaryMessage {
		// in protocol v4 & binary msg ps: {"type":0,"data":{"sid":"HWEr440000:1:R1CHyink:shadiao:101"},"nsp":"/","id":0}
		c.out <- &protocol.MsgPack{
			Type: protocol.CONNECT,
			Nsp:  protocol.DefaultNsp,
			Data: struct {
				Sid string `json:"sid"`
			}{Sid: c.Id()},
		}
	} else {
		// GET /socket.io/?EIO=4&transport=polling&t=N8hyd7H&sid=lv_VI97HAXpY6yYWAAAC
		// < HTTP/1.1 200 OK
		// < Content-Type: text/plain; charset=UTF-8
		// 40
		// in protocol v4 & text msg ps: 0{"sid":"DJehCG0000:1:07d8SFHH:shadiao:101"}
		marshal, err := json.Marshal(&struct {
			Sid string `json:"sid"`
		}{
			Sid: c.Id(),
		})
		if err != nil {
			panic(err)
		}

		c.out <- protocol.CommonMsg + protocol.OpenMsg + string(marshal)
	}
}

/**
Setup event loop for given connection
*/
func (s *Server) SetupEventLoop(conn *websocket.Connection, remoteAddr string,
	r *http.Request) {

	interval, timeout := conn.PingParams()
	hdr := Header{
		Sid:          generateNewId(remoteAddr),
		Upgrades:     []string{},
		PingInterval: int(interval / time.Millisecond),
		PingTimeout:  int(timeout / time.Millisecond),
	}

	c := &Channel{}
	c.conn = conn
	c.ip = remoteAddr
	c.request = r
	c.initChannel()

	c.server = s
	c.header = hdr

	s.SendOpenSequence(c)

	go inLoop(c, &s.methods)
	go outLoop(c, &s.methods)

	s.callLoopEvent(c, OnConnection)
}

/**
implements ServeHTTP function from http.Handler
*/
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for key, el := range s.headers {
		w.Header().Set(key, el)
	}

	conn, err := s.tr.HandleConnection(w, r)
	if err != nil {
		return
	}

	s.SetupEventLoop(conn, r.RemoteAddr, r)
	s.tr.Serve(w, r)
}

/**
Get amount of current connected sids
*/
func (s *Server) AmountOfSids() int64 {
	s.sidsLock.RLock()
	defer s.sidsLock.RUnlock()

	return int64(len(s.sids))
}

/**
Get amount of rooms with at least one channel(or sid) joined
*/
func (s *Server) AmountOfRooms() int64 {
	s.channelsLock.RLock()
	defer s.channelsLock.RUnlock()

	return int64(len(s.channels))
}

func (s *Server) EnableCORS(domain string) {
	s.headers["Access-Control-Allow-Origin"] = domain
	s.headers["Access-Control-Allow-Credentials"] = "true"
}

func (s *Server) AddHeader(name string, value string) {
	s.headers[name] = value
}

func (s *Server) UpdateTransport(tr websocket.Transport) {
	s.tr = tr
}

func NewServer(tr websocket.Transport) *Server {
	s := Server{}
	s.tr = tr
	s.headers = make(map[string]string)
	s.channels = make(map[string]map[*Channel]struct{})
	s.rooms = make(map[*Channel]map[string]struct{})
	s.sids = make(map[string]*Channel)
	s.onConnection = onConnectStore
	s.onDisconnection = onDisconnectCleanup

	return &s
}
