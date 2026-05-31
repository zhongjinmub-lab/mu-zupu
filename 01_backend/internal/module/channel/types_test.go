package channel

import "testing"

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
