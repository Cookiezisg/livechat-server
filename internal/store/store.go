package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

const (
	MessageTypeText   = "text"
	MessageTypeAction = "action"
	MessageTypeSystem = "system"
)

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *Store) CreateUser(ctx context.Context, email, username, passwordHash, avatarURL string) (User, error) {
	id := uuid.New()
	query := `
		insert into users (id, email, username, password_hash, avatar_url)
		values ($1, $2, $3, $4, $5)
		returning id, email, username, avatar_url, created_at`
	var user User
	err := s.db.QueryRow(ctx, query, id, email, username, passwordHash, avatarURL).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.AvatarURL,
		&user.CreatedAt,
	)
	return user, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, string, error) {
	query := `select id, email, username, avatar_url, password_hash, created_at from users where lower(email) = lower($1)`
	var user User
	var passwordHash string
	err := s.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.AvatarURL,
		&passwordHash,
		&user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", ErrNotFound
	}
	return user, passwordHash, err
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	query := `select id, email, username, avatar_url, created_at from users where id = $1`
	var user User
	err := s.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.AvatarURL,
		&user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (s *Store) SearchUsers(ctx context.Context, query string, limit int) ([]User, error) {
	q := `%` + query + `%`
	rows, err := s.db.Query(ctx, `
		select id, email, username, avatar_url, created_at
		from users
		where username ilike $1 or email ilike $1
		order by username
		limit $2`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.Username, &user.AvatarURL, &user.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, user)
	}
	return out, rows.Err()
}

func (s *Store) CreateRoom(ctx context.Context, name string, isPrivate bool, createdBy uuid.UUID) (Room, error) {
	id := uuid.New()
	query := `
		insert into rooms (id, name, is_private, created_by)
		values ($1, $2, $3, $4)
		returning id, name, is_private, created_by, created_at`
	var room Room
	err := s.db.QueryRow(ctx, query, id, name, isPrivate, createdBy).Scan(
		&room.ID,
		&room.Name,
		&room.IsPrivate,
		&room.CreatedBy,
		&room.CreatedAt,
	)
	return room, err
}

func (s *Store) ListRooms(ctx context.Context, userID uuid.UUID) ([]Room, error) {
	rows, err := s.db.Query(ctx, `
		select r.id, r.name, r.is_private, r.created_by, r.created_at
		from rooms r
		left join room_members m on r.id = m.room_id and m.user_id = $1
		where r.is_private = false or m.user_id is not null
		order by r.created_at desc`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Room
	for rows.Next() {
		var room Room
		if err := rows.Scan(&room.ID, &room.Name, &room.IsPrivate, &room.CreatedBy, &room.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, room)
	}
	return out, rows.Err()
}

func (s *Store) GetRoom(ctx context.Context, roomID uuid.UUID) (Room, error) {
	var room Room
	query := `select id, name, is_private, created_by, created_at from rooms where id = $1`
	err := s.db.QueryRow(ctx, query, roomID).Scan(
		&room.ID,
		&room.Name,
		&room.IsPrivate,
		&room.CreatedBy,
		&room.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Room{}, ErrNotFound
	}
	return room, err
}

func (s *Store) AddRoomMember(ctx context.Context, roomID, userID uuid.UUID, role string) error {
	_, err := s.db.Exec(ctx, `
		insert into room_members (room_id, user_id, role)
		values ($1, $2, $3)
		on conflict (room_id, user_id) do nothing`, roomID, userID, role)
	return err
}

func (s *Store) RemoveRoomMember(ctx context.Context, roomID, userID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `delete from room_members where room_id = $1 and user_id = $2`, roomID, userID)
	return err
}

func (s *Store) IsRoomMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `select exists(select 1 from room_members where room_id = $1 and user_id = $2)`, roomID, userID).Scan(&exists)
	return exists, err
}

func (s *Store) UpdateRoomReadAt(ctx context.Context, roomID, userID uuid.UUID, readAt time.Time) error {
	_, err := s.db.Exec(ctx, `
		update room_members set last_read_at = $1 where room_id = $2 and user_id = $3`, readAt, roomID, userID)
	return err
}

func (s *Store) CreateMessage(ctx context.Context, roomID *uuid.UUID, senderID uuid.UUID, recipientID *uuid.UUID, body, attachmentURL, messageType string) (Message, error) {
	id := uuid.New()
	query := `
		insert into messages (id, room_id, sender_id, recipient_id, body, attachment_url, message_type)
		values ($1, $2, $3, $4, $5, $6, $7)
		returning id, room_id, sender_id, recipient_id, body, attachment_url, message_type, created_at`
	var msg Message
	err := s.db.QueryRow(ctx, query, id, roomID, senderID, recipientID, body, attachmentURL, messageType).Scan(
		&msg.ID,
		&msg.RoomID,
		&msg.SenderID,
		&msg.RecipientID,
		&msg.Body,
		&msg.AttachmentURL,
		&msg.MessageType,
		&msg.CreatedAt,
	)
	return msg, err
}

func (s *Store) ListRoomMessages(ctx context.Context, roomID uuid.UUID, limit int, before time.Time) ([]Message, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		select m.id, m.room_id, m.sender_id, m.recipient_id, m.body, m.attachment_url, m.message_type, m.created_at,
		       u.username, u.avatar_url
		from messages m
		join users u on u.id = m.sender_id
		where m.room_id = $1 and m.created_at < $2
		order by m.created_at desc
		limit $3`, roomID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

func (s *Store) ListDMs(ctx context.Context, userA, userB uuid.UUID, limit int, before time.Time) ([]Message, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		select m.id, m.room_id, m.sender_id, m.recipient_id, m.body, m.attachment_url, m.message_type, m.created_at,
		       u.username, u.avatar_url
		from messages m
		join users u on u.id = m.sender_id
		where m.room_id is null
		  and ((m.sender_id = $1 and m.recipient_id = $2) or (m.sender_id = $2 and m.recipient_id = $1))
		  and m.created_at < $3
		order by m.created_at desc
		limit $4`, userA, userB, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (User, error) {
	var user User
	err := s.db.QueryRow(ctx, `select id, email, username, avatar_url, created_at from users where lower(username) = lower($1)`, username).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.AvatarURL,
		&user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

func scanMessages(rows pgx.Rows) ([]Message, error) {
	var out []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(
			&msg.ID,
			&msg.RoomID,
			&msg.SenderID,
			&msg.RecipientID,
			&msg.Body,
			&msg.AttachmentURL,
			&msg.MessageType,
			&msg.CreatedAt,
			&msg.SenderName,
			&msg.SenderAvatar,
		); err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) Health(ctx context.Context) error {
	return s.db.Ping(ctx)
}

func (s *Store) EnsureDefaultRoom(ctx context.Context) (Room, error) {
	var room Room
	query := `select id, name, is_private, created_by, created_at from rooms where lower(name) = lower($1)`
	err := s.db.QueryRow(ctx, query, "general").Scan(
		&room.ID,
		&room.Name,
		&room.IsPrivate,
		&room.CreatedBy,
		&room.CreatedAt,
	)
	if err == nil {
		return room, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Room{}, err
	}
	id := uuid.New()
	creator := uuid.New()
	insert := `insert into rooms (id, name, is_private, created_by) values ($1, $2, false, $3)`
	if _, err := s.db.Exec(ctx, insert, id, "general", creator); err != nil {
		return Room{}, fmt.Errorf("seed default room: %w", err)
	}
	return s.GetRoom(ctx, id)
}
