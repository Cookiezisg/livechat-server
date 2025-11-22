package chat

import (
	"sort"
	"strings"
	"sync"
)

// Room 表示单个聊天室，会管理成员列表、话题以及最近的消息历史。
type Room struct {
	key          string
	name         string
	topic        string
	mu           sync.RWMutex
	members      map[string]*Client
	history      []string
	historyLimit int
}

// newRoom 根据传入名字创建房间实例，同时确保名称和历史上限可用。
func newRoom(name string, historyLimit int) *Room {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = "lobby"
	}
	return &Room{
		key:          normalizeRoomKey(trimmed),
		name:         trimmed,
		members:      make(map[string]*Client),
		history:      make([]string, 0, historyLimit),
		historyLimit: historyLimit,
	}
}

// Key 返回房间的内部唯一键，用于 Server 中的 map 查询。
func (r *Room) Key() string {
	return r.key
}

// Name 返回用户可见的房间名称。
func (r *Room) Name() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.name
}

// SetName 更新房间名称，空值会被忽略。
func (r *Room) SetName(name string) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return
	}
	r.mu.Lock()
	r.name = trimmed
	r.mu.Unlock()
}

// Topic 返回房间当前的话题内容。
func (r *Room) Topic() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.topic
}

// SetTopic 设置房间话题并返回最新的值。
func (r *Room) SetTopic(topic string) string {
	trimmed := strings.TrimSpace(topic)
	r.mu.Lock()
	r.topic = trimmed
	r.mu.Unlock()
	return trimmed
}

// AddMember 将客户端加入成员表。
func (r *Room) AddMember(c *Client) {
	r.mu.Lock()
	r.members[c.id] = c
	r.mu.Unlock()
}

// RemoveMember 从成员表中移除指定 ID。
func (r *Room) RemoveMember(id string) {
	r.mu.Lock()
	delete(r.members, id)
	r.mu.Unlock()
}

// MemberCount 返回房间当前在线人数。
func (r *Room) MemberCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.members)
}

// IsEmpty 判断房间是否没有任何成员。
func (r *Room) IsEmpty() bool {
	return r.MemberCount() == 0
}

// MembersSnapshot 复制成员列表，可排除某个客户端，以便在锁外广播消息。
func (r *Room) MembersSnapshot(exclude *Client) []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Client, 0, len(r.members))
	for _, member := range r.members {
		if exclude != nil && member == exclude {
			continue
		}
		list = append(list, member)
	}
	return list
}

// MemberNames 返回按字母序排列的成员昵称。
func (r *Room) MemberNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.members))
	for _, member := range r.members {
		names = append(names, member.DisplayName())
	}
	sort.Strings(names)
	return names
}

// AppendHistory 将消息存入房间历史并裁剪超出部分。
func (r *Room) AppendHistory(entry string) {
	if r.historyLimit <= 0 {
		return
	}
	r.mu.Lock()
	r.history = append(r.history, entry)
	if len(r.history) > r.historyLimit {
		r.history = r.history[len(r.history)-r.historyLimit:]
	}
	r.mu.Unlock()
}

// RecentHistory 返回最近 limit 条历史的副本，供新加入的用户查看。
func (r *Room) RecentHistory(limit int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit > len(r.history) {
		limit = len(r.history)
	}
	start := len(r.history) - limit
	if start < 0 {
		start = 0
	}
	out := make([]string, limit)
	copy(out, r.history[start:])
	return out
}
