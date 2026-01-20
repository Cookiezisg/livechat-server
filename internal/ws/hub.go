package ws

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"livechat-server/internal/auth"
	"livechat-server/internal/store"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Hub struct {
	store     *store.Store
	secret    string
	register  chan *Conn
	unregister chan *Conn
	broadcast chan outboundMessage
	mu        sync.RWMutex
	users     map[uuid.UUID]map[*Conn]bool
	rooms     map[uuid.UUID]map[*Conn]bool
}

type outboundMessage struct {
	roomID  *uuid.UUID
	userID  *uuid.UUID
	event   Event
	exclude *Conn
}

func NewHub(store *store.Store, secret string) *Hub {
	return &Hub{
		store:      store,
		secret:     secret,
		register:   make(chan *Conn),
		unregister: make(chan *Conn),
		broadcast:  make(chan outboundMessage, 128),
		users:      make(map[uuid.UUID]map[*Conn]bool),
		rooms:      make(map[uuid.UUID]map[*Conn]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.addConn(conn)
		case conn := <-h.unregister:
			h.removeConn(conn)
		case msg := <-h.broadcast:
			h.send(msg)
		}
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := middlewareExtractToken(r)
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	claims, err := auth.ParseToken(token, h.secret)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn := NewConn(wsConn, h, user)
	h.register <- conn
	go conn.writeLoop()
	conn.readLoop()
}

func (h *Hub) addConn(conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.users[conn.user.ID]; !ok {
		h.users[conn.user.ID] = make(map[*Conn]bool)
	}
	h.users[conn.user.ID][conn] = true
	h.broadcastPresenceLocked(conn.user.ID, true)
}

func (h *Hub) removeConn(conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	userConns := h.users[conn.user.ID]
	if userConns != nil {
		delete(userConns, conn)
		if len(userConns) == 0 {
			delete(h.users, conn.user.ID)
			h.broadcastPresenceLocked(conn.user.ID, false)
		}
	}
	for roomID, conns := range h.rooms {
		if conns[conn] {
			delete(conns, conn)
			if len(conns) == 0 {
				delete(h.rooms, roomID)
			}
		}
	}
}

func (h *Hub) send(msg outboundMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if msg.roomID != nil {
		for conn := range h.rooms[*msg.roomID] {
			if msg.exclude != nil && conn == msg.exclude {
				continue
			}
			conn.enqueue(msg.event)
		}
		return
	}
	if msg.userID != nil {
		for conn := range h.users[*msg.userID] {
			if msg.exclude != nil && conn == msg.exclude {
				continue
			}
			conn.enqueue(msg.event)
		}
	}
}

func (h *Hub) SubscribeRoom(roomID uuid.UUID, conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.rooms[roomID]; !ok {
		h.rooms[roomID] = make(map[*Conn]bool)
	}
	h.rooms[roomID][conn] = true
}

func (h *Hub) UnsubscribeRoom(roomID uuid.UUID, conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns := h.rooms[roomID]
	if conns == nil {
		return
	}
	delete(conns, conn)
	if len(conns) == 0 {
		delete(h.rooms, roomID)
	}
}

func (h *Hub) broadcastRoom(roomID uuid.UUID, event Event, exclude *Conn) {
	h.broadcast <- outboundMessage{roomID: &roomID, event: event, exclude: exclude}
}

func (h *Hub) broadcastUser(userID uuid.UUID, event Event, exclude *Conn) {
	h.broadcast <- outboundMessage{userID: &userID, event: event, exclude: exclude}
}

func (h *Hub) broadcastPresenceLocked(userID uuid.UUID, online bool) {
	payload := PresencePayload{UserID: userID, Online: online, LastSeen: time.Now().UTC()}
	data := Event{Type: "presence", Data: payload}
	for conn := range h.allConnectionsLocked() {
		conn.enqueue(data)
	}
}

func (h *Hub) allConnectionsLocked() map[*Conn]bool {
	all := make(map[*Conn]bool)
	for _, conns := range h.users {
		for conn := range conns {
			all[conn] = true
		}
	}
	return all
}

func middlewareExtractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}
	return ""
}
