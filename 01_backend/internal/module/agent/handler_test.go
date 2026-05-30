package agent

import (
	"testing"

	"mu-agent-saas/internal/module/kb"
)

func TestConversationTitleTruncatesRunes(t *testing.T) {
	title := conversationTitle("  这是一个用于测试会话标题截断的长消息  ")
	if title != "这是一个用于测试会话标题截断的长消息" {
		t.Fatalf("title = %q", title)
	}

	long := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	title = conversationTitle(long)
	if len([]rune(title)) != 80 {
		t.Fatalf("title rune length = %d", len([]rune(title)))
	}
}

func TestBuildChatMessagesUsesSystemHistoryAndCurrentUserMessage(t *testing.T) {
	messages := buildChatMessages(
		Agent{SystemPrompt: "system prompt"},
		[]kb.SearchResult{{Title: "doc", Content: "knowledge"}},
		[]Message{
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "second"},
			{Role: "tool", Content: "ignored"},
		},
		"current",
	)
	if len(messages) != 4 {
		t.Fatalf("message count = %d", len(messages))
	}
	if messages[0].Role != "system" || messages[1].Content != "first" || messages[2].Content != "second" || messages[3].Content != "current" {
		t.Fatalf("messages = %#v", messages)
	}
}
