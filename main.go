// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net/http"
	"websockt-char/service"
)
//设定参数
var addr = flag.String("addr", ":8080", "http service address")

//聊天室中间件
func ServeHome(w http.ResponseWriter, r *http.Request) {
	//打印访问日志
	log.Println(r.URL)
	//检验路径是否正确
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	//检验方法是否为GET
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	//绑定html文件
	http.ServeFile(w, r, "home.html")
}

func main() {
	//解析参数
	flag.Parse()
	hub := service.NewHub()
	go hub.Run()
	//聊天室接口，聊天室界面
	http.HandleFunc("/", ServeHome)
	//监听接口，监听用户发送的信息，并将信息写入响应体里面
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		service.ServeWs(hub, w, r)
	})
	//监听8080端口信息
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}