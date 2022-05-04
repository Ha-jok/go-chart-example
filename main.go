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

var addr = flag.String("addr", ":8080", "http service address")

//聊天室中间件
func ServeHome(w http.ResponseWriter, r *http.Request) {
	//输出访问日志
	log.Println(r.URL,"abc")
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
	flag.Parse()
	hub := service.NewHub()
	go hub.Run()
	http.HandleFunc("/", ServeHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		service.ServeWs(hub, w, r)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}