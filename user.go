package main

import (
	"fmt"
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

// 创建一个用户的API
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()

	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		C:      make(chan string),
		Conn:   conn,
		Server: server,
	}

	go user.ListenMessage()

	return user
}

// 监听当前User channel的方法，一旦有消息，就直接发给对端客户端
func (u *User) ListenMessage() {

	for {
		msg := <-u.C

		u.Conn.Write([]byte(msg + "\n"))
	}
}

// 用户上线的业务
func (u *User) Online() {
	// 用户上线，将用户加入到onlineMap中
	u.Server.mapLock.Lock()
	u.Server.OnlineMap[u.Name] = u
	u.Server.mapLock.Unlock()

	// 广播当前用户上线消息
	u.Server.BroadCast(u, "was online")
}

// 用户下线的业务
func (u *User) Offline() {
	// 用户下线，将用户加入到OfflineMap中
	u.Server.mapLock.Lock()
	delete(u.Server.OnlineMap, u.Name)
	u.Server.mapLock.Unlock()

	// 广播当前用户上线消息
	u.Server.BroadCast(u, "was offline")
}

// 用户下线的业务
func (u *User) SendMsg(msg string) {
	u.Conn.Write([]byte(msg))
}

// 用户处理消息业务
func (u *User) DoMessage(msg string) {
	if msg == "who" {
		// 查询当前在线用户有哪些
		u.Server.mapLock.Lock()
		for _, user := range u.Server.OnlineMap {
			onlineMsg := fmt.Sprintf("[%s]%s is Online..\n", user.Addr, user.Name)
			u.SendMsg(onlineMsg)
		}
		u.Server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		// 消息格式： rename|张三
		newName := strings.Split(msg, "|")[1]

		// 判断name是否存在
		_, ok := u.Server.OnlineMap[newName]
		if ok {
			u.SendMsg("Current username are taken\n")
		} else {
			u.Server.mapLock.Lock()
			delete(u.Server.OnlineMap, u.Name)
			u.Server.OnlineMap[newName] = u
			u.Server.mapLock.Unlock()

			u.Name = newName
			u.SendMsg(fmt.Sprintf("Your name have changed: %s\n", u.Name))
		}
	} else if len(msg) > 4 && msg[:3] == "to|" {
		// 消息格式： to|张三|消息内容

		// 1 获取对方用户名
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			u.SendMsg("Message format error, pelse use to|zhangsan|content\n")
			return
		}

		// 2 根据用户名，得到对方user对象
		remoteUser, ok := u.Server.OnlineMap[remoteName]
		if !ok {
			u.SendMsg("User isn't exist\n")
			return
		}

		// 3 获取消息内容，通过对方user对象发送过去
		content := strings.Split(msg, "|")[2]
		if content == "" {
			u.SendMsg("No content, please try again\n")
			return
		}
		remoteUser.SendMsg(fmt.Sprintf("%s send msg to you:%s", u.Name, content))
	} else {
		u.Server.BroadCast(u, msg)
	}
}
