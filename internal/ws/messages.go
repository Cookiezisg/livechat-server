package ws

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type ClientEvent struct {
	Type          string `json:"type"`
	RoomID        string `json:"roomId,omitempty"`
	RecipientID   string `json:"recipientId,omitempty"`
	Body          string `json:"body,omitempty"`
	AttachmentURL string `json:"attachmentUrl,omitempty"`
	IsTyping      bool   `json:"isTyping,omitempty"`
}

type MessagePayload struct {
	ID            uuid.UUID  `json:"id"`
	RoomID        *uuid.UUID `json:"roomId,omitempty"`
	SenderID      uuid.UUID  `json:"senderId"`
	SenderName    string     `json:"senderName"`
	SenderAvatar  string     `json:"senderAvatar"`
	RecipientID   *uuid.UUID `json:"recipientId,omitempty"`
	Body          string     `json:"body"`
	AttachmentURL string     `json:"attachmentUrl,omitempty"`
	MessageType   string     `json:"messageType"`
	CreatedAt     time.Time  `json:"createdAt"`
}

type PresencePayload struct {
	UserID   uuid.UUID `json:"userId"`
	Online   bool      `json:"online"`
	LastSeen time.Time `json:"lastSeen"`
}

type TypingPayload struct {
	RoomID      *uuid.UUID `json:"roomId,omitempty"`
	RecipientID *uuid.UUID `json:"recipientId,omitempty"`
	UserID      uuid.UUID  `json:"userId"`
	IsTyping    bool       `json:"isTyping"`
}

type RoomJoinPayload struct {
	RoomID uuid.UUID `json:"roomId"`
	UserID uuid.UUID `json:"userId"`
}
