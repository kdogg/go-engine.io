package websocket

import (
	"io"
	"net/http"

	"fmt"
	"github.com/googollee/go-engine.io/message"
	"github.com/googollee/go-engine.io/parser"
	"github.com/googollee/go-engine.io/transport"
	"github.com/gorilla/websocket"

	"sync" // KK
)

const KK = true

type Server struct {
	mut sync.Mutex // KK

	callback transport.Callback
	conn     *websocket.Conn
}

func NewServer(w http.ResponseWriter, r *http.Request, callback transport.Callback) (transport.Server, error) {
	conn, err := websocket.Upgrade(w, r, nil, 10240, 10240)
	if err != nil {
		return nil, err
	}

	fmt.Printf("****************************  engine.io UPGRADED to websocket ***************************\n")

	ret := &Server{

		mut: sync.Mutex{}, // KK

		callback: callback,
		conn:     conn,
	}

	go ret.serveHTTP(w, r)

	return ret, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
}

func (s *Server) NextWriter(msgType message.MessageType, packetType parser.PacketType) (io.WriteCloser, error) {
	wsType, newEncoder := websocket.TextMessage, parser.NewStringEncoder
	if msgType == message.MessageBinary {
		wsType, newEncoder = websocket.BinaryMessage, parser.NewBinaryEncoder
	}

	w, err := func() (io.WriteCloser, error) { // KK
		s.mut.Lock()
		defer s.mut.Unlock()
		w, err := s.conn.NextWriter(wsType)
		return w, err
	}()

	if err != nil {
		return nil, err
	}
	ret, err := newEncoder(w, packetType)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *Server) Close() error {
	if KK {
		s.mut.Lock()
		defer s.mut.Unlock()
	}

	return s.conn.Close()
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	defer s.callback.OnClose(s)

	for {
		t, r, err := s.conn.NextReader()
		if err != nil {
			return
		}

		switch t {
		case websocket.TextMessage:
			fallthrough
		case websocket.BinaryMessage:
			decoder, err := parser.NewDecoder(r)
			if err != nil {
				return
			}
			s.callback.OnPacket(decoder)
			decoder.Close()
		}
	}
}
