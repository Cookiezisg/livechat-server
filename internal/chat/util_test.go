package chat

import "testing"

// util_test.go 覆盖与用户/房间名字解析相关的工具函数。

func TestSanitizeUserName(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"Alice", false},
		{"bob_123", false},
		{"xy", true},
		{"this-name-is-way-too-long-for-the-server", true},
		{"bad name", true},
	}
	for _, tt := range tests {
		_, err := sanitizeUserName(tt.input)
		if tt.wantErr && err == nil {
			t.Fatalf("sanitizeUserName(%q) expected error", tt.input)
		}
		if !tt.wantErr && err != nil {
			t.Fatalf("sanitizeUserName(%q) unexpected error: %v", tt.input, err)
		}
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input string
		cmd   string
		args  string
	}{
		{"/nick Alice", "nick", "Alice"},
		{"/rooms", "rooms", ""},
		{" /MSG  bob   hello there ", "msg", "bob   hello there"},
	}
	for _, tt := range tests {
		cmd, args := parseCommand(tt.input)
		if cmd != tt.cmd || args != tt.args {
			t.Fatalf("parseCommand(%q) = (%q,%q) want (%q,%q)", tt.input, cmd, args, tt.cmd, tt.args)
		}
	}
}

func TestNormalizeRoomKey(t *testing.T) {
	if got := normalizeRoomKey("  GoLang  "); got != "golang" {
		t.Fatalf("normalizeRoomKey failed, got %q", got)
	}
}
