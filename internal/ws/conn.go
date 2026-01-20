package ws

import (
	"context"
	"strings"
	"time"

	"livechat-server/internal/store"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Conn struct {
	socket *websocket.Conn
	hub    *Hub
	user   store.User
	send   chan Event
	rooms  map[uuid.UUID]bool
}

func NewConn(socket *websocket.Conn, hub *Hub, user store.User) *Conn {
	return &Conn{
		socket: socket,
		hub:    hub,
		user:   user,
		send:   make(chan Event, 64),
		rooms:  make(map[uuid.UUID]bool),
	}
}

func (c *Conn) readLoop() {
	defer func() {
		c.hub.unregister <- c
		_ = c.socket.Close()
	}()
	for {
		var msg ClientEvent
		if err := c.socket.ReadJSON(&msg); err != nil {
			return
		}
		c.handleClientEvent(msg)
	}
}

func (c *Conn) writeLoop() {
	defer func() {
		_ = c.socket.Close()
	}()
	for event := range c.send {
		if err := c.socket.WriteJSON(event); err != nil {
			return
		}
	}
}

func (c *Conn) enqueue(event Event) {
	select {
	case c.send <- event:
	default:
	}
}

func (c *Conn) handleClientEvent(msg ClientEvent) {
	switch msg.Type {
	case "join_room":
		c.joinRoom(msg.RoomID)
	case "leave_room":
		c.leaveRoom(msg.RoomID)
	case "message":
		c.handleMessage(msg)
	case "typing":
		c.handleTyping(msg)
	case "read":
		c.handleRead(msg)
	}
}

func (c *Conn) joinRoom(roomID string) {
	parsed, err := uuid.Parse(roomID)
	if err != nil {
		c.enqueue(Event{Type: "error", Data: map[string]string{"message": "invalid room"}})
		return
	}
	_ = c.hub.store.AddRoomMember(context.Background(), parsed, c.user.ID, "member")
	c.hub.SubscribeRoom(parsed, c)
	c.rooms[parsed] = true
	c.enqueue(Event{Type: "room_joined", Data: RoomJoinPayload{RoomID: parsed, UserID: c.user.ID}})
}

func (c *Conn) leaveRoom(roomID string) {
	parsed, err := uuid.Parse(roomID)
	if err != nil {
		return
	}
	_ = c.hub.store.RemoveRoomMember(context.Background(), parsed, c.user.ID)
	delete(c.rooms, parsed)
	c.hub.UnsubscribeRoom(parsed, c)
}

func (c *Conn) handleMessage(msg ClientEvent) {
	body := strings.TrimSpace(msg.Body)
	if body == "" && msg.AttachmentURL == "" {
		return
	}
	if msg.RoomID != "" {
		c.sendRoomMessage(msg, body)
		return
	}
	if msg.RecipientID != "" {
		c.sendDMMessage(msg, body)
	}
}

func (c *Conn) sendRoomMessage(msg ClientEvent, body string) {
	roomID, err := uuid.Parse(msg.RoomID)
	if err != nil {
		return
	}
	member, err := c.hub.store.IsRoomMember(context.Background(), roomID, c.user.ID)
	if err != nil || !member {
		c.enqueue(Event{Type: "error", Data: map[string]string{"message": "not a member"}})
		return
	}
	message, err := c.hub.store.CreateMessage(context.Background(), &roomID, c.user.ID, nil, body, msg.AttachmentURL, store.MessageTypeText)
	if err != nil {
		return
	}
	payload := MessagePayload{
		ID:            message.ID,
		RoomID:        &roomID,
		SenderID:      c.user.ID,
		SenderName:    c.user.Username,
		SenderAvatar:  c.user.AvatarURL,
		RecipientID:   nil,
		Body:          message.Body,
		AttachmentURL: message.AttachmentURL,
		MessageType:   message.MessageType,
		CreatedAt:     message.CreatedAt,
	}
	c.hub.broadcastRoom(roomID, Event{Type: "message", Data: payload}, nil)
}

func (c *Conn) sendDMMessage(msg ClientEvent, body string) {
	recipientID, err := uuid.Parse(msg.RecipientID)
	if err != nil {
		return
	}
	message, err := c.hub.store.CreateMessage(context.Background(), nil, c.user.ID, &recipientID, body, msg.AttachmentURL, store.MessageTypeText)
	if err != nil {
		return
	}
	payload := MessagePayload{
		ID:            message.ID,
		RoomID:        nil,
		SenderID:      c.user.ID,
		SenderName:    c.user.Username,
		SenderAvatar:  c.user.AvatarURL,
		RecipientID:   &recipientID,
		Body:          message.Body,
		AttachmentURL: message.AttachmentURL,
		MessageType:   message.MessageType,
		CreatedAt:     message.CreatedAt,
	}
	c.hub.broadcastUser(recipientID, Event{Type: "message", Data: payload}, nil)
	c.hub.broadcastUser(c.user.ID, Event{Type: "message", Data: payload}, nil)
}

func (c *Conn) handleTyping(msg ClientEvent) {
	payload := TypingPayload{
		UserID:   c.user.ID,
		IsTyping: msg.IsTyping,
	}
	if msg.RoomID != "" {
		roomID, err := uuid.Parse(msg.RoomID)
		if err != nil {
			return
		}
		payload.RoomID = &roomID
		c.hub.broadcastRoom(roomID, Event{Type: "typing", Data: payload}, c)
		return
	}
	if msg.RecipientID != "" {
		recipientID, err := uuid.Parse(msg.RecipientID)
		if err != nil {
			return
		}
		payload.RecipientID = &recipientID
		c.hub.broadcastUser(recipientID, Event{Type: "typing", Data: payload}, c)
	}
}

func (c *Conn) handleRead(msg ClientEvent) {
	if msg.RoomID == "" {
		return
	}
	roomID, err := uuid.Parse(msg.RoomID)
	if err != nil {
		return
	}
	_ = c.hub.store.UpdateRoomReadAt(context.Background(), roomID, c.user.ID, time.Now().UTC())
}
