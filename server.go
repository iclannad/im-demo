package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	// 在线用户列表
	OnlineMap map[string]*User
	mapLock   sync.RWMutex

	// 消息广播的channel
	Message chan string
}

// 创建一个server的接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

// 监听Message广播消息channel的goroutine，一旦有消息就发送给全部在线的用户
func (s *Server) ListenMessager() {
	for {
		msg := <-s.Message

		// 将message发送给全部在线用户
		s.mapLock.Lock()
		for _, cli := range s.OnlineMap {
			cli.C <- msg
		}
		s.mapLock.Unlock()
	}
}

// 广播消息的方法
func (s *Server) BroadCast(user *User, msg string) {
	sendMsg := fmt.Sprintf("[%s]%s:%s", user.Addr, user.Name, msg)

	s.Message <- sendMsg
}

func (s *Server) Handler(conn net.Conn) {
	//...当前链接的业务
	fmt.Println("链接建立成功")

	user := NewUser(conn, s)

	// 用户上线，将用户加入到onlineMap中
	user.Online()

	// 监听用户是否活跃的channel
	isLive := make(chan bool)

	// 接受客户端发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err:", err)
				return
			}
			msg := string(buf[:n])
			user.DoMessage(msg)

			// 客户发送的任意消息,代表当前用户是一个活跃的状态
			isLive <- true
		}
	}()

	for {
		select {
		case <-isLive:
		case <-time.After(time.Minute * 10):
			// 当前用户已经超时
			user.SendMsg("You are kicked out\n")
			close(user.C)
			conn.Close()
			return
		}
	}

}

// 启动服务器接口
func (s *Server) Start() {
	// socket listen
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Ip, s.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	// close listen socket
	defer listen.Close()

	go s.ListenMessager()

	for {
		// accept
		accept, err := listen.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue
		}

		// do handler
		go s.Handler(accept)
	}
}
