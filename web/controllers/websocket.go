// Copyright 2013 Beego Samples authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package controllers

import (
	"context"
	"io"
	"time"

	"ehang.io/nps/lib/common"
	"ehang.io/nps/lib/pmux"
	"ehang.io/nps/server/connection"
	"github.com/astaxie/beego"
	"nhooyr.io/websocket"
)

// WebSocketController handles WebSocket requests.
type WebSocketController struct {
	beego.Controller
}

// Get method handles GET requests for WebSocketController.
func (s *WebSocketController) Get() {
	scope := s.GetString("scope")
	// Upgrade from http request to WebSocket.
	ws, err := websocket.Accept(s.Ctx.ResponseWriter, s.Ctx.Request, nil)
	if err != nil {
		if scope == "ws" {
			s.TplName = "ws/websocket.html"
			s.Data["IsWebSocket"] = false
		} else {
			s.Redirect("/", 302)
		}
		return
	}
	defer ws.Close(websocket.StatusInternalError, "the server is falling")
	ctx, cancel := context.WithTimeout(s.Ctx.Request.Context(), time.Hour*12)
	defer cancel()
	conn := websocket.NetConn(ctx, ws, websocket.MessageText)
	buf := make([]byte, 3)
	if n, err := io.ReadFull(conn, buf); err != nil || n != 3 {
		conn.Close()
		return
	}
	if common.BytesToNum(buf) == pmux.CLIENT {
		ch := connection.GlobalPMux.GetClientConn()
		timer := time.NewTimer(pmux.ACCEPT_TIME_OUT)
		eventClose := make(chan bool)
		select {
		case <-timer.C:
		case ch <- pmux.NewPortConn(conn, buf, false, eventClose):
		}
		<-eventClose
	}
	conn.Close()
}
