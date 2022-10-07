package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Server 结构体
// 通过ip和port登录
type Server struct {
	Ip        string
	Port      int
	OnlineMap map[string]*User
	mapLock   sync.RWMutex
	Message   chan string
}

func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

func (server *Server) Start() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", server.Ip, server.Port))
	if err != nil {
		fmt.Println("net listen err:", err)
		return
	}
	defer listener.Close()

	go server.ListenMessager()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue
		}

		go server.Handler(conn)
	}
}

func (server *Server) Handler(conn net.Conn) {
	// fmt.Println("连接成功...")
	// 创建当前用户
	user := NewUser(conn, server)

	// // 写入map
	// server.mapLock.Lock()
	// server.OnlineMap[user.Name] = user
	// server.mapLock.Unlock()

	// // 广播当前用户上线的消息
	// server.BroadCast(user, "已上线")
	user.Online()

	isLive := make(chan bool)
	// 用户消息广播
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				// server.BroadCast(user, "下线...")
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("conn read err:", err)
				return
			}

			msg := string(buf[:n-1])

			// server.BroadCast(user, msg)
			user.DoMessage(msg)

			isLive <- true
		}
	}()

	for {
		select {
		case <-isLive:
		case <-time.After(time.Second * 100):
			user.SendMsg("你被踢了")
			close(user.C)
			conn.Close()
			return
		}
	}
}

func (server *Server) BroadCast(user *User, msg string) {
	sendMeg := "[" + user.Addr + "]" + user.Name + ":" + msg + "\n"
	server.Message <- sendMeg
}

// 监听Message广播消息channel的goroutinue，一旦有消息就发送给全部的在线的User
func (server *Server) ListenMessager() {
	for {
		msg := <-server.Message

		server.mapLock.Lock()
		for _, cli := range server.OnlineMap {
			cli.C <- msg
		}
		server.mapLock.Unlock()
	}
}
