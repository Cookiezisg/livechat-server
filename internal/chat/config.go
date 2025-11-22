// Package chat 提供聊天室核心的会话、房间与客户端管理能力。
package chat

import "time"

// Config 描述聊天服务器的运行参数，可通过命令行或默认值生成。
type Config struct {
	IdleTimeout     time.Duration // 客户端多久无操作会被断开
	HistoryLimit    int           // 每个房间最多保留多少条历史
	MaxMessageBytes int           // 单条消息允许的最大字节数
	LobbyName       string        // 默认大厅房间名
	SystemName      string        // 系统提示前缀
	SendBufferSize  int           // 客户端发送缓冲区大小
}

// DefaultConfig 返回可供练手使用的默认配置。
func DefaultConfig() Config {
	return Config{
		IdleTimeout:     2 * time.Minute,
		HistoryLimit:    50,
		MaxMessageBytes: 4096,
		LobbyName:       "lobby",
		SystemName:      "livechat",
		SendBufferSize:  64,
	}
}

// withDefaults 把缺失或非法的字段填充成默认值，保证运行稳定。
func (c Config) withDefaults() Config {
	defaults := DefaultConfig()
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = defaults.IdleTimeout
	}
	if c.HistoryLimit <= 0 {
		c.HistoryLimit = defaults.HistoryLimit
	}
	if c.MaxMessageBytes <= 0 {
		c.MaxMessageBytes = defaults.MaxMessageBytes
	}
	if c.LobbyName == "" {
		c.LobbyName = defaults.LobbyName
	}
	if c.SystemName == "" {
		c.SystemName = defaults.SystemName
	}
	if c.SendBufferSize <= 0 {
		c.SendBufferSize = defaults.SendBufferSize
	}
	return c
}
