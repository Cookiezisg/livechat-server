package chat

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// Client 表示一个通过 TCP 连接到服务器的用户会话，负责收发消息以及状态维护。
type Client struct {
	id     string
	conn   net.Conn
	server *Server

	send      chan string
	done      chan struct{}
	idleReset chan struct{}
	once      sync.Once

	mu          sync.RWMutex
	name        string
	roomKey     string
	roomDisplay string
}

// newClient 创建并初始化一个客户端结构体实例。
func newClient(server *Server, conn net.Conn, id, name string) *Client {
	return &Client{
		id:        id,
		conn:      conn,
		server:    server,
		send:      make(chan string, server.cfg.SendBufferSize),
		done:      make(chan struct{}),
		idleReset: make(chan struct{}, 1),
		name:      name,
	}
}

// start 启动客户端生命周期：异步写循环、超时监听以及同步读取输入。
func (c *Client) start() {
	go c.writeLoop()
	go c.idleWatcher()
	c.sendSystem(fmt.Sprintf("Welcome to %s! Type /help for commands.", c.server.cfg.SystemName))
	c.server.joinRoom(c, c.server.cfg.LobbyName, true)
	c.readLoop()
}

// readLoop 使用 bufio.Scanner 持续读取用户输入，并在每次成功读取后重置空闲计时器。
func (c *Client) readLoop() {
	defer c.shutdown("connection closed")
	scanner := bufio.NewScanner(c.conn)
	scanner.Buffer(make([]byte, 0, 1024), c.server.cfg.MaxMessageBytes)
	for scanner.Scan() {
		line := scanner.Text()
		c.touch()
		c.server.handleInput(c, line)
	}
	if err := scanner.Err(); err != nil && !isNetClosed(err) {
		c.server.logf("read error from %s: %v", c.DisplayName(), err)
	}
}

// writeLoop 负责把 send 通道中的消息写入 TCP 连接，必要时限制超时时间。
func (c *Client) writeLoop() {
	for {
		select {
		case <-c.done:
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if !strings.HasSuffix(msg, "\n") {
				msg += "\n"
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
			if _, err := io.WriteString(c.conn, msg); err != nil {
				if !isNetClosed(err) {
					c.server.logf("write error to %s: %v", c.DisplayName(), err)
				}
				c.shutdown("write failure")
				return
			}
		}
	}
}

// idleWatcher 监控客户端是否长时间无动作，超时后主动断开连接。
func (c *Client) idleWatcher() {
	if c.server.cfg.IdleTimeout <= 0 {
		return
	}
	timer := time.NewTimer(c.server.cfg.IdleTimeout)
	for {
		select {
		case <-c.done:
			timer.Stop()
			return
		case <-timer.C:
			c.sendSystem("Disconnected due to inactivity.")
			c.shutdown("idle timeout")
			return
		case <-c.idleReset:
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(c.server.cfg.IdleTimeout)
		}
	}
}

// Send 将文本写入发送缓冲，若缓冲已满则自动丢弃并记录日志。
func (c *Client) Send(message string) {
	sanitized := strings.TrimRight(message, "\r\n")
	select {
	case <-c.done:
		return
	default:
	}
	select {
	case c.send <- sanitized:
	default:
		c.server.logf("send buffer full for %s; dropping message", c.DisplayName())
	}
}

// sendSystem 是 Send 的包装，会自动附加系统前缀。
func (c *Client) sendSystem(message string) {
	c.Send(fmt.Sprintf("[%s] %s", c.server.cfg.SystemName, message))
}

// shutdown 保证只执行一次的清理流程：关闭通道、连接并通知 Server。
func (c *Client) shutdown(reason string) {
	c.once.Do(func() {
		close(c.done)
		close(c.send)
		_ = c.conn.Close()
		c.server.removeClient(c, reason)
	})
}

// touch 尝试重置空闲计时器，若通道已满则静默忽略。
func (c *Client) touch() {
	select {
	case c.idleReset <- struct{}{}:
	default:
	}
}

// Name 安全地返回客户端当前昵称。
func (c *Client) Name() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.name
}

// setName 更新昵称，仅供服务端在完成校验后调用。
func (c *Client) setName(name string) {
	c.mu.Lock()
	c.name = name
	c.mu.Unlock()
}

// DisplayName 在当前实现等同于 Name，预留进一步拼装信息的扩展点。
func (c *Client) DisplayName() string {
	return c.Name()
}

// RoomKey 返回客户端所在房间的内部键值。
func (c *Client) RoomKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.roomKey
}

// RoomName 返回客户端所在房间对用户可见的名称。
func (c *Client) RoomName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.roomDisplay
}

// setRoom 更新客户端缓存的房间信息。
func (c *Client) setRoom(room *Room) {
	c.mu.Lock()
	c.roomKey = room.Key()
	c.roomDisplay = room.Name()
	c.mu.Unlock()
}

// isNetClosed 判断给定错误是否意味着 TCP 连接已经关闭。
func isNetClosed(err error) bool {
	return errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF)
}
