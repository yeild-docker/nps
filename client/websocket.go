package client

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
	"nhooyr.io/websocket"
)

var (
	CachedWebsocketConn sync.Map
)

type WebsocketConn struct {
	ConnType string
	Conn     *websocket.Conn
	NetConn  net.Conn
	WsCtx    context.Context
	WsCancel context.CancelFunc
}

func NewWebsocketConn(ConnType string, conn *websocket.Conn, connection net.Conn, ctx context.Context, cancel context.CancelFunc) *WebsocketConn {
	return &WebsocketConn{
		ConnType: ConnType,
		Conn:     conn,
		NetConn:  connection,
		WsCtx:    ctx,
		WsCancel: cancel,
	}
}

func (s *WebsocketConn) WsHeartBeat(d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	for {
		select {
		case <-s.WsCtx.Done():
			goto stop
		case <-t.C:
		}
		logs.Debug("%s Ping", s.ConnType)
		err := s.Conn.Ping(s.WsCtx)
		if err != nil {
			goto stop
		}
		logs.Debug("%s Pong", s.ConnType)
		t.Reset(d)
	}
stop:
	logs.Debug("%s exited", s.ConnType)
	CloseWebSocketConn(s.ConnType)
}

func LoadWebSocketConn(connType string) *WebsocketConn {
	if v, ok := CachedWebsocketConn.Load(connType); ok {
		return v.(*WebsocketConn)
	}
	return nil
}

func CacheWebSocketConn(connType string, conn *WebsocketConn) {
	CachedWebsocketConn.Store(connType, conn)
}

func CloseWebSocketConn(connType string) {
	s := LoadWebSocketConn(connType)
	if s == nil {
		return
	}
	if s.WsCancel != nil {
		s.WsCancel()
	}
	s.WsCtx = nil
	s.WsCancel = nil
	CachedWebsocketConn.Delete(s.ConnType)
}

func DialWebsocketConn(server string, connType string) (net.Conn, error) {
	if conn := LoadWebSocketConn(connType); conn != nil {
		return conn.NetConn, nil
	}
	wsCtx, wsCancel := context.WithTimeout(context.Background(), time.Hour*12)
	var ws *websocket.Conn
	ws, _, err := websocket.Dial(wsCtx, server, nil)
	if err != nil {
		return nil, err
	}
	netConn := websocket.NetConn(wsCtx, ws, websocket.MessageText)
	wsConn := NewWebsocketConn(connType, ws, netConn, wsCtx, wsCancel)
	CacheWebSocketConn(connType, wsConn)
	go wsConn.WsHeartBeat(30 * time.Second)
	return netConn, nil
}
