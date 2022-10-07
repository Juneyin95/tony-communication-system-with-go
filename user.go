package main

import (
	"net"
	"strings"
)

type User struct {
	Name   string
	Addr   string
	C      chan string
	Conn   net.Conn
	Server *Server
}

func NewUser(conn net.Conn, server *Server) *User {
	username := conn.RemoteAddr().String()

	user := &User{
		Name:   username,
		Addr:   username,
		C:      make(chan string),
		Conn:   conn,
		Server: server,
	}

	go user.ListenMessage()

	return user
}

func (user *User) ListenMessage() {
	for {
		msg := <-user.C
		// user.Conn.Write([]byte(msg + "\n"))
		user.SendMsg(msg)
	}
}

func (user *User) SendMsg(msg string) {
	user.Conn.Write([]byte(msg))
}

func (user *User) Online() {
	// 写入map
	user.Server.mapLock.Lock()
	user.Server.OnlineMap[user.Name] = user
	user.Server.mapLock.Unlock()

	// 广播当前用户上线的消息
	user.Server.BroadCast(user, "已上线")
}

func (user *User) Offline() {
	// 从map中删除
	user.Server.mapLock.Lock()
	delete(user.Server.OnlineMap, user.Name)
	user.Server.mapLock.Unlock()

	user.Server.BroadCast(user, "下线...")
}

func (user *User) DoMessage(msg string) {
	if msg == "who" {
		user.Server.mapLock.Lock()
		for _, u := range user.Server.OnlineMap {
			onlineMsg := "[" + u.Addr + "]" + u.Name + ":" + "在线...\n"
			user.SendMsg(onlineMsg)
		}
		user.Server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		newName := strings.Split(msg, "|")[1]
		_, ok := user.Server.OnlineMap[newName]
		if ok {
			user.SendMsg("该用户名已经存在，请重新修改....\n")
		} else {
			user.Server.mapLock.Lock()
			delete(user.Server.OnlineMap, user.Name)
			user.Server.OnlineMap[newName] = user
			user.Server.mapLock.Unlock()
			user.Name = newName
			user.SendMsg("已经修改成功" + user.Name + "\n")
		}
	} else if len(msg) > 4 && msg[:3] == "to|" {
		// to|张三|msg
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			user.SendMsg("输入格式为：to|张三|nihao，请重新输入...\n")
			return
		}
		_, ok := user.Server.OnlineMap[remoteName]
		if !ok {
			user.SendMsg("该用户不存在，请重新输入...\n")
			return
		}
		content := strings.Split(msg, "|")[2]
		if content == "" {
			user.SendMsg("输入消息不能为空，请重新输入...\n")
			return
		}
		sendContent := user.Name + "说：" + content + "\n"
		user.Server.OnlineMap[remoteName].SendMsg(sendContent)
	} else {
		user.Server.BroadCast(user, msg)
	}
}
