package chat

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Server 负责管理所有房间、客户端以及指令路由，是聊天系统的核心。
type Server struct {
	cfg Config

	mu      sync.RWMutex
	rooms   map[string]*Room
	clients map[string]*Client
	names   map[string]*Client

	nextID uint64
}

// NewServer 根据传入配置创建一个聊天服务器实例。
func NewServer(cfg Config) *Server {
	cfg = cfg.withDefaults()
	return &Server{
		cfg:     cfg,
		rooms:   make(map[string]*Room),
		clients: make(map[string]*Client),
		names:   make(map[string]*Client),
	}
}

// ListenAndServe 启动 TCP 监听，并在上下文取消之前持续接受新连接。
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			return err
		}
		go s.handleConn(conn)
	}
}

// handleConn 为单条 TCP 连接创建 Client 并进入会话流程。
func (s *Server) handleConn(conn net.Conn) {
	seq := atomic.AddUint64(&s.nextID, 1)
	name := fmt.Sprintf("guest-%04d", seq)
	id := fmt.Sprintf("client-%d", seq)
	client := newClient(s, conn, id, name)
	if err := s.addClient(client); err != nil {
		s.logf("failed to register client: %v", err)
		_ = conn.Close()
		return
	}
	s.logf("client %s connected from %s", client.DisplayName(), conn.RemoteAddr())
	client.start()
}

// addClient 将新客户端注册到在线列表并确保昵称唯一。
func (s *Server) addClient(c *Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.ToLower(c.Name())
	if _, exists := s.names[key]; exists {
		return fmt.Errorf("username %q already in use", c.Name())
	}
	s.clients[c.id] = c
	s.names[key] = c
	return nil
}

// removeClient 在客户端断开后做清理并通知房间内其他成员。
func (s *Server) removeClient(c *Client, reason string) {
	s.mu.Lock()
	delete(s.clients, c.id)
	delete(s.names, strings.ToLower(c.Name()))
	roomKey := c.RoomKey()
	var room *Room
	if roomKey != "" {
		room = s.rooms[roomKey]
		if room != nil {
			room.RemoveMember(c.id)
			if room.IsEmpty() && !strings.EqualFold(room.Name(), s.cfg.LobbyName) {
				delete(s.rooms, roomKey)
			}
		}
	}
	s.mu.Unlock()

	if room != nil {
		s.broadcastRoom(room, fmt.Sprintf("%s left (%s)", c.DisplayName(), reason), false, c)
	}
	s.logf("client %s disconnected: %s", c.DisplayName(), reason)
}

// joinRoom 让客户端进入（或创建）指定房间，并选择是否广播。
func (s *Server) joinRoom(c *Client, desired string, announce bool) {
	roomName := desired
	if strings.TrimSpace(roomName) == "" {
		roomName = s.cfg.LobbyName
	}
	validName, err := sanitizeRoomName(roomName)
	if err != nil {
		c.sendSystem(err.Error())
		return
	}
	room, previous, already := s.moveClientRoom(c, validName)
	if already {
		c.sendSystem(fmt.Sprintf("You are already in #%s", room.Name()))
		return
	}

	c.sendSystem(fmt.Sprintf("Joined #%s (%d users)", room.Name(), room.MemberCount()))
	if topic := room.Topic(); topic != "" {
		c.sendSystem(fmt.Sprintf("Topic: %s", topic))
	}
	history := room.RecentHistory(s.cfg.HistoryLimit)
	if len(history) > 0 {
		c.sendSystem(fmt.Sprintf("Recent %d messages:", len(history)))
		for _, line := range history {
			c.Send(fmt.Sprintf("[%s] %s", room.Name(), line))
		}
	}

	if announce {
		s.broadcastRoom(room, fmt.Sprintf("%s joined the room", c.DisplayName()), false, c)
	}
	if previous != nil && previous != room {
		s.broadcastRoom(previous, fmt.Sprintf("%s left the room", c.DisplayName()), false, c)
		s.cleanupRoom(previous)
	}
}

// moveClientRoom 完成房间映射更新，返回当前房间、之前房间以及是否已有。
func (s *Server) moveClientRoom(c *Client, roomName string) (*Room, *Room, bool) {
	key := normalizeRoomKey(roomName)
	if key == "" {
		key = normalizeRoomKey(s.cfg.LobbyName)
	}

	s.mu.Lock()
	room, ok := s.rooms[key]
	if !ok {
		room = newRoom(roomName, s.cfg.HistoryLimit)
		s.rooms[key] = room
	}
	currentKey := c.RoomKey()
	if currentKey == room.Key() {
		s.mu.Unlock()
		return room, room, true
	}
	var previous *Room
	if currentKey != "" {
		previous = s.rooms[currentKey]
		if previous != nil {
			previous.RemoveMember(c.id)
		}
	}
	room.AddMember(c)
	c.setRoom(room)
	s.mu.Unlock()
	return room, previous, false
}

// cleanupRoom 在房间为空且非大厅时删除它，避免泄露。
func (s *Server) cleanupRoom(room *Room) {
	if room == nil || strings.EqualFold(room.Name(), s.cfg.LobbyName) {
		return
	}
	if !room.IsEmpty() {
		return
	}
	s.mu.Lock()
	if room.IsEmpty() {
		delete(s.rooms, room.Key())
	}
	s.mu.Unlock()
}

// handleInput 将客户端原始输入分流到命令或普通消息。
func (s *Server) handleInput(c *Client, line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	if strings.HasPrefix(trimmed, "/") {
		cmd, args := parseCommand(trimmed)
		if cmd == "" {
			return
		}
		s.executeCommand(c, cmd, args)
		return
	}
	s.postMessage(c, trimmed)
}

// executeCommand 解析好的命令会在这里分派到对应处理函数。
func (s *Server) executeCommand(c *Client, cmd, args string) {
	switch cmd {
	case "help":
		c.sendSystem(helpText)
	case "nick":
		s.cmdNick(c, args)
	case "who":
		s.cmdWho(c, args)
	case "rooms":
		s.cmdRooms(c)
	case "join":
		s.joinRoom(c, args, true)
	case "leave":
		s.joinRoom(c, s.cfg.LobbyName, true)
	case "msg":
		s.cmdMsg(c, args)
	case "topic":
		s.cmdTopic(c, args)
	case "history":
		s.cmdHistory(c, args)
	case "me":
		s.cmdAction(c, args)
	case "quit", "exit":
		c.sendSystem("Bye!")
		c.shutdown("quit")
	case "whoami":
		c.sendSystem(fmt.Sprintf("You are %s in #%s", c.DisplayName(), c.RoomName()))
	case "ping":
		c.sendSystem("pong")
	default:
		c.sendSystem("Unknown command. Type /help for assistance.")
	}
}

// postMessage 把普通文本广播到当前房间，并持久化历史。
func (s *Server) postMessage(c *Client, text string) {
	room := s.roomForClient(c)
	if room == nil {
		c.sendSystem("You are not inside a room. Use /join <room> first.")
		return
	}
	s.broadcastRoom(room, fmt.Sprintf("%s: %s", c.DisplayName(), text), true, nil)
}

// cmdAction 处理 /me 动作，生成带格式的房间广播。
func (s *Server) cmdAction(c *Client, args string) {
	if strings.TrimSpace(args) == "" {
		c.sendSystem("Usage: /me <action>")
		return
	}
	room := s.roomForClient(c)
	if room == nil {
		c.sendSystem("You are not inside a room. Use /join <room> first.")
		return
	}
	s.broadcastRoom(room, fmt.Sprintf("* %s %s", c.DisplayName(), args), true, nil)
}

// cmdNick 处理改名请求，确保名称合规且唯一。
func (s *Server) cmdNick(c *Client, args string) {
	if strings.TrimSpace(args) == "" {
		c.sendSystem("Usage: /nick <new-name>")
		return
	}
	name, err := sanitizeUserName(args)
	if err != nil {
		c.sendSystem(err.Error())
		return
	}
	oldName := c.DisplayName()

	s.mu.Lock()
	if _, exists := s.names[strings.ToLower(name)]; exists {
		s.mu.Unlock()
		c.sendSystem("That name is already taken.")
		return
	}
	delete(s.names, strings.ToLower(oldName))
	s.names[strings.ToLower(name)] = c
	s.mu.Unlock()

	c.setName(name)
	c.sendSystem(fmt.Sprintf("You're now known as %s", name))
	if room := s.roomForClient(c); room != nil {
		s.broadcastRoom(room, fmt.Sprintf("%s is now known as %s", oldName, name), false, c)
	}
}

// cmdWho 根据参数返回当前房间或全部在线用户。
func (s *Server) cmdWho(c *Client, args string) {
	if strings.EqualFold(strings.TrimSpace(args), "all") {
		s.mu.RLock()
		names := make([]string, 0, len(s.clients))
		for _, client := range s.clients {
			names = append(names, client.DisplayName())
		}
		s.mu.RUnlock()
		sort.Strings(names)
		c.sendSystem(fmt.Sprintf("Online (%d): %s", len(names), strings.Join(names, ", ")))
		return
	}
	room := s.roomForClient(c)
	if room == nil {
		c.sendSystem("You are not inside a room.")
		return
	}
	names := room.MemberNames()
	c.sendSystem(fmt.Sprintf("#%s (%d): %s", room.Name(), len(names), strings.Join(names, ", ")))
}

// cmdRooms 输出房间列表及其在线人数、话题。
func (s *Server) cmdRooms(c *Client) {
	s.mu.RLock()
	type summary struct {
		name  string
		count int
		topic string
	}
	summaries := make([]summary, 0, len(s.rooms))
	for _, room := range s.rooms {
		summaries = append(summaries, summary{name: room.Name(), count: room.MemberCount(), topic: room.Topic()})
	}
	s.mu.RUnlock()

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].count == summaries[j].count {
			return summaries[i].name < summaries[j].name
		}
		return summaries[i].count > summaries[j].count
	})
	if len(summaries) == 0 {
		c.sendSystem("No active rooms yet. Be the first to /join one!")
		return
	}
	builder := strings.Builder{}
	builder.WriteString("Rooms:\n")
	for _, sum := range summaries {
		line := fmt.Sprintf(" - #%s (%d)", sum.name, sum.count)
		if sum.topic != "" {
			line += fmt.Sprintf(" – %s", sum.topic)
		}
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	c.Send(builder.String())
}

// cmdMsg 实现 /msg 私信能力。
func (s *Server) cmdMsg(c *Client, args string) {
	parts := strings.Fields(args)
	if len(parts) < 2 {
		c.sendSystem("Usage: /msg <user> <message>")
		return
	}
	targetName := parts[0]
	message := strings.TrimSpace(args[len(targetName):])
	if strings.TrimSpace(message) == "" {
		c.sendSystem("You need to provide a message body.")
		return
	}
	s.directMessage(c, targetName, message)
}

// directMessage 向指定用户发送点对点消息，并在双方客户端展示。
func (s *Server) directMessage(from *Client, targetName, text string) {
	s.mu.RLock()
	target := s.names[strings.ToLower(targetName)]
	s.mu.RUnlock()
	if target == nil {
		from.sendSystem("User not found.")
		return
	}
	payload := strings.TrimSpace(text)
	if payload == "" {
		from.sendSystem("Message cannot be empty.")
		return
	}
	toMsg := fmt.Sprintf("[dm] %s → you: %s", from.DisplayName(), payload)
	fromMsg := fmt.Sprintf("[dm] you → %s: %s", target.DisplayName(), payload)
	target.Send(toMsg)
	from.Send(fromMsg)
}

// cmdTopic 查看或设置房间话题。
func (s *Server) cmdTopic(c *Client, args string) {
	room := s.roomForClient(c)
	if room == nil {
		c.sendSystem("You are not inside a room.")
		return
	}
	if strings.TrimSpace(args) == "" {
		if topic := room.Topic(); topic == "" {
			c.sendSystem("No topic set for this room.")
		} else {
			c.sendSystem(fmt.Sprintf("Topic for #%s: %s", room.Name(), topic))
		}
		return
	}
	topic := room.SetTopic(args)
	s.broadcastRoom(room, fmt.Sprintf("%s set the topic: %s", c.DisplayName(), topic), false, nil)
}

// cmdHistory 取最近 N 条房间历史给调用方。
func (s *Server) cmdHistory(c *Client, args string) {
	room := s.roomForClient(c)
	if room == nil {
		c.sendSystem("You are not inside a room.")
		return
	}
	limit, err := parseOptionalInt(args, 10)
	if err != nil {
		c.sendSystem("History length must be a number.")
		return
	}
	if limit <= 0 {
		limit = 10
	}
	history := room.RecentHistory(limit)
	if len(history) == 0 {
		c.sendSystem("No history yet.")
		return
	}
	c.sendSystem(fmt.Sprintf("Last %d messages in #%s:", len(history), room.Name()))
	for _, line := range history {
		c.Send(fmt.Sprintf("[%s] %s", room.Name(), line))
	}
}

// broadcastRoom 将消息发送给房间内所有成员，必要时写入历史。
func (s *Server) broadcastRoom(room *Room, text string, store bool, exclude *Client) {
	if room == nil {
		return
	}
	if store {
		room.AppendHistory(text)
	}
	recipients := room.MembersSnapshot(exclude)
	formatted := fmt.Sprintf("[%s] %s", room.Name(), text)
	for _, member := range recipients {
		member.Send(formatted)
	}
}

// roomForClient 根据客户端缓存的房间键查找实例。
func (s *Server) roomForClient(c *Client) *Room {
	key := c.RoomKey()
	if key == "" {
		return nil
	}
	s.mu.RLock()
	room := s.rooms[key]
	s.mu.RUnlock()
	return room
}

// logf 为模块化日志输出加上 chat 前缀。
func (s *Server) logf(format string, args ...any) {
	log.Printf("[chat] "+format, args...)
}

// helpText 列出所有可用指令，发送给 /help 调用者。
const helpText = `Available commands:
/Help                Show this message
/Who [all]           List users in current room or all rooms
/Rooms               List all rooms
/Join <room>         Join or create a room
/Leave               Return to the lobby
/Nick <name>         Change your display name
/Msg <user> <text>   Send a private message
/Topic [text]        View or set the room topic
/History [n]         Show the last n messages from the room
/Me <action>         Share a role-play style action
/Whoami              Display your current identity
/Ping                Check connection health
/Quit                Disconnect from the server`
