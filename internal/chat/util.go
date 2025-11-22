package chat

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// normalizeRoomKey 将房间名转为内部统一的键（去空格 + 小写）。
func normalizeRoomKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// sanitizeRoomName 校验房间名称是否合法并返回规范化结果。
func sanitizeRoomName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("room name cannot be empty")
	}
	if len(trimmed) > 32 {
		return "", fmt.Errorf("room name is too long (max 32 characters)")
	}
	for _, r := range trimmed {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == ' ') {
			return "", fmt.Errorf("invalid character %q in room name", r)
		}
	}
	return trimmed, nil
}

// sanitizeUserName 校验用户昵称长度与字符集是否合法。
func sanitizeUserName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if len(trimmed) < 3 {
		return "", fmt.Errorf("name must be at least 3 characters long")
	}
	if len(trimmed) > 24 {
		return "", fmt.Errorf("name is too long (max 24 characters)")
	}
	for _, r := range trimmed {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_') {
			return "", fmt.Errorf("invalid character %q in name", r)
		}
	}
	return trimmed, nil
}

// parseCommand 把以 / 开头的命令拆成命令名与余下参数。
func parseCommand(raw string) (string, string) {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return "", ""
	}
	parts := strings.SplitN(trimmed, " ", 2)
	cmd := strings.ToLower(parts[0])
	if len(parts) == 1 {
		return cmd, ""
	}
	return cmd, strings.TrimSpace(parts[1])
}

// parseOptionalInt 尝试解析一个可选整数，不填时返回默认值。
func parseOptionalInt(value string, fallback int) (int, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return n, nil
}
