package channel

import (
	"strings"
	"testing"
)

func TestDefaultChannelTypesInstallableFlag(t *testing.T) {
	byType := map[string]ChannelType{}
	for _, def := range DefaultChannelTypes() {
		byType[def.Type] = def
	}
	if !byType["web"].Installable || byType["web"].Status != "active" {
		t.Fatalf("web channel should be active and installable, got %#v", byType["web"])
	}
	if byType["wechat_official"].Installable || byType["wechat_official"].Status != "planned" {
		t.Fatalf("wechat_official should be planned and not installable, got %#v", byType["wechat_official"])
	}
}

func TestFindChannelType(t *testing.T) {
	if _, ok := FindChannelType("  WEB "); !ok {
		t.Fatal("expected to find web ignoring case and spaces")
	}
	if _, ok := FindChannelType("telegram"); ok {
		t.Fatal("expected unknown channel type to be not found")
	}
}

func TestCreateChannelRequestNormalizeAndValidate(t *testing.T) {
	req := CreateChannelRequest{AgentID: "  a1 ", Type: "  WEB ", Name: "  官网客服 "}
	req.Normalize()
	if req.AgentID != "a1" || req.Type != "web" || req.Name != "官网客服" {
		t.Fatalf("normalized request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request: %v", err)
	}
}

func TestCreateChannelRequestRejectsUnavailableType(t *testing.T) {
	req := CreateChannelRequest{AgentID: "a1", Type: "wechat_official", Name: "公众号"}
	req.Normalize()
	if err := req.Validate(); err == nil {
		t.Fatal("expected error for not-yet-available channel type")
	}

	missingAgent := CreateChannelRequest{Type: "web", Name: "x"}
	missingAgent.Normalize()
	if err := missingAgent.Validate(); err == nil {
		t.Fatal("expected error when agent_id is empty")
	}

	unknown := CreateChannelRequest{AgentID: "a1", Type: "telegram", Name: "x"}
	unknown.Normalize()
	if err := unknown.Validate(); err == nil {
		t.Fatal("expected error for unknown channel type")
	}
}

func TestBuildChannelEmbedWeb(t *testing.T) {
	ch := Channel{Type: "web", Status: StatusEnabled, ChannelKey: "ch_abc"}
	embed := BuildChannelEmbed(ch, "https://demo.example.com/")
	if !embed.Enabled {
		t.Fatal("enabled web channel should report enabled")
	}
	if embed.APIEndpoint != "https://demo.example.com/api/v1/channels/ch_abc/chat" {
		t.Fatalf("unexpected api endpoint: %s", embed.APIEndpoint)
	}
	if !strings.Contains(embed.EmbedSnippet, "ch_abc") || !strings.Contains(embed.EmbedSnippet, "<script") {
		t.Fatalf("web snippet should contain script with key: %s", embed.EmbedSnippet)
	}
}

func TestBuildChannelEmbedAPIIncludesCurl(t *testing.T) {
	ch := Channel{Type: "api", Status: StatusEnabled, ChannelKey: "ch_x"}
	embed := BuildChannelEmbed(ch, "http://localhost:8080")
	if !strings.HasPrefix(embed.EmbedSnippet, "curl -X POST ") {
		t.Fatalf("api snippet should be a curl example: %s", embed.EmbedSnippet)
	}
}

func TestBuildChannelEmbedDisabledAddsHint(t *testing.T) {
	ch := Channel{Type: "web", Status: StatusDisabled, ChannelKey: "ch_y"}
	embed := BuildChannelEmbed(ch, "https://demo.example.com")
	if embed.Enabled {
		t.Fatal("disabled channel should not be enabled")
	}
	var hinted bool
	for _, line := range embed.Instructions {
		if strings.Contains(line, "未启用") {
			hinted = true
		}
	}
	if !hinted {
		t.Fatalf("disabled channel should include a not-enabled hint: %#v", embed.Instructions)
	}
}

func TestUpdateChannelRequestNormalizeAndValidate(t *testing.T) {
	req := UpdateChannelRequest{Name: "  新名称  "}
	req.Normalize()
	if req.Name != "新名称" {
		t.Fatalf("normalized name = %q", req.Name)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid update: %v", err)
	}

	// 仅更新配置也应通过。
	configOnly := UpdateChannelRequest{Config: map[string]any{"theme": "dark"}}
	configOnly.Normalize()
	if err := configOnly.Validate(); err != nil {
		t.Fatalf("config-only update should be valid: %v", err)
	}

	// 名称与配置都为空应报错。
	empty := UpdateChannelRequest{}
	empty.Normalize()
	if err := empty.Validate(); err == nil {
		t.Fatal("expected error when neither name nor config provided")
	}
}

func TestSummarizeChannels(t *testing.T) {
	channels := []Channel{
		{Type: "web", Status: StatusEnabled},
		{Type: "web", Status: StatusDisabled},
		{Type: "api", Status: StatusEnabled},
	}
	summary := SummarizeChannels(channels)
	if summary.Total != 3 || summary.Enabled != 2 || summary.Disabled != 1 {
		t.Fatalf("unexpected summary counts: %#v", summary)
	}
	if summary.ByType["web"] != 2 || summary.ByType["api"] != 1 {
		t.Fatalf("unexpected by-type distribution: %#v", summary.ByType)
	}
}

func TestSummarizeChannelsEmpty(t *testing.T) {
	summary := SummarizeChannels(nil)
	if summary.Total != 0 || len(summary.ByType) != 0 {
		t.Fatalf("empty summary should be zero-valued: %#v", summary)
	}
}
