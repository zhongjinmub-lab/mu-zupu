package agent

import (
	"strings"
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

func TestSplitAnswerDeltasKeepsRuneBoundaries(t *testing.T) {
	answer := strings.Repeat("智", 61)
	parts := splitAnswerDeltas(answer)
	if len(parts) != 2 {
		t.Fatalf("parts count = %d", len(parts))
	}
	if len([]rune(parts[0])) != 60 || len([]rune(parts[1])) != 1 {
		t.Fatalf("unexpected rune lengths: %d %d", len([]rune(parts[0])), len([]rune(parts[1])))
	}
	if parts[0]+parts[1] != answer {
		t.Fatalf("parts do not reconstruct original answer")
	}
}

func TestSplitAnswerDeltasReturnsEmptyDeltaForBlankAnswer(t *testing.T) {
	parts := splitAnswerDeltas("   ")
	if len(parts) != 1 || parts[0] != "" {
		t.Fatalf("parts = %#v", parts)
	}
}

func TestDefaultToolSafetyPolicyDefaultsToDeny(t *testing.T) {
	policy := DefaultToolSafetyPolicy()
	if policy.Enabled || policy.DefaultAction != "deny" || !policy.DangerConfirmation {
		t.Fatalf("unexpected tool policy: %#v", policy)
	}
	if len(policy.DangerousTools) == 0 || !policy.DangerousTools[0].RequiresConfirmation {
		t.Fatalf("expected dangerous tools to require confirmation: %#v", policy.DangerousTools)
	}
	if policy.AuditAction != "agent.tool.call" || policy.PermissionRole == "" {
		t.Fatalf("expected audit and permission metadata: %#v", policy)
	}
}
