// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package service

import (
	"bytes"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)
//定义基础信息，
const (
	// Time allowed to write a message to the peer.
	//允许写信息的时长
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	//允许读取下一条pong信息的时长
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	//在此期间发送ping信息
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	//信息最大的大小
	maxMessageSize = 512
)
//声明换行符和空格
var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)
//升级协议相关配置
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
//定义客户端的结构
type Client struct {
	hub *Hub

	// The websocket connection.
	//TCP的连接
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	//定义一个通道
	send chan []byte
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
//读取信息
func (c *Client) readPump() {
	//延时关闭连接。
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	//对连接信息通道设置限制，
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	//从连接通道里面读取信息
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		//将换行符替换为空格
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		//将读取的信息传入广播通道
		c.hub.broadcast <- message
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
//发送信息
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	//延时关闭连接
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	//建立一个永久循环对信息通道和时间进行读取检验
	for {
		select {
		case message, ok := <-c.send:
			//设置时间限制
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				//通道写入状态码
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			//将信息写入请求响应
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			//将send里面的信息读取到websockt中
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
//监听中间件
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	//升级协议
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	//读取用户信息
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	//开启读写服务
	go client.writePump()
	go client.readPump()
}
