package store

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	AvatarURL string    `json:"avatarUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

type Room struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	IsPrivate bool      `json:"isPrivate"`
	CreatedBy uuid.UUID `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
}

type Message struct {
	ID            uuid.UUID  `json:"id"`
	RoomID        *uuid.UUID `json:"roomId,omitempty"`
	SenderID      uuid.UUID  `json:"senderId"`
	RecipientID   *uuid.UUID `json:"recipientId,omitempty"`
	Body          string     `json:"body"`
	MessageType   string     `json:"messageType"`
	AttachmentURL string     `json:"attachmentUrl,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	SenderName    string     `json:"senderName"`
	SenderAvatar  string     `json:"senderAvatar"`
}

type RoomMember struct {
	RoomID     uuid.UUID `json:"roomId"`
	UserID     uuid.UUID `json:"userId"`
	Role       string    `json:"role"`
	JoinedAt   time.Time `json:"joinedAt"`
	LastReadAt time.Time `json:"lastReadAt"`
}
