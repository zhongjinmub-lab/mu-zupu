package tenant

import (
	"testing"
	"time"
)

func TestAddMemberRequestNormalizeAndValidate(t *testing.T) {
	req := AddMemberRequest{Email: " USER@Example.COM ", RoleCode: ""}
	req.Normalize()
	if req.Email != "user@example.com" || req.RoleCode != "member" {
		t.Fatalf("normalized request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request: %v", err)
	}
}

func TestUpdateMemberRoleRequestValidateRejectsInvalidRole(t *testing.T) {
	req := UpdateMemberRoleRequest{RoleCode: "root"}
	req.Normalize()
	if err := req.Validate(); err == nil {
		t.Fatal("expected role validation error")
	}
}

func TestCanManageMembers(t *testing.T) {
	if !CanManageMembers("owner") || !CanManageMembers("admin") {
		t.Fatal("owner/admin should manage members")
	}
	if CanManageMembers("member") || CanManageMembers("viewer") {
		t.Fatal("member/viewer should not manage members")
	}
}

func TestRolePermissionsSummarizesRoles(t *testing.T) {
	summary := RolePermissions("admin")
	if summary.CurrentRole != "admin" {
		t.Fatalf("current role = %q", summary.CurrentRole)
	}
	if len(summary.Roles) != 4 {
		t.Fatalf("roles length = %d", len(summary.Roles))
	}
	var owner, viewer RolePermission
	for _, role := range summary.Roles {
		switch role.RoleCode {
		case "owner":
			owner = role
		case "viewer":
			viewer = role
		}
	}
	if !owner.CanManage || !owner.CanWrite || !owner.CanRead {
		t.Fatalf("unexpected owner permissions: %#v", owner)
	}
	if viewer.CanWrite || viewer.CanManage || !viewer.CanRead {
		t.Fatalf("unexpected viewer permissions: %#v", viewer)
	}
}

func TestCreateInvitationRequestNormalizeAndValidate(t *testing.T) {
	req := CreateInvitationRequest{Email: " Invite@Example.COM ", RoleCode: "owner"}
	req.Normalize()
	if req.Email != "invite@example.com" || req.RoleCode != "admin" || req.TTLHours != 168 {
		t.Fatalf("normalized invitation request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid invitation request: %v", err)
	}
}

func TestCreateInvitationRequestRejectsInvalidTTL(t *testing.T) {
	req := CreateInvitationRequest{Email: "invite@example.com", RoleCode: "member", TTLHours: 721}
	req.Normalize()
	if err := req.Validate(); err == nil {
		t.Fatal("expected ttl validation error")
	}
}

func TestInvitationTokenHash(t *testing.T) {
	token, hash, err := newInvitationToken()
	if err != nil {
		t.Fatalf("new token: %v", err)
	}
	if token == "" || hash == "" || token == hash {
		t.Fatalf("token/hash not generated correctly")
	}
	if hashInvitationToken(token) != hash {
		t.Fatal("hash should be stable")
	}
}

func TestAuditLogQueryNormalizeAndValidate(t *testing.T) {
	q := AuditLogQuery{
		TenantID:     " tenant-1 ",
		ActorUserID:  "7b7d45a7-10f7-4aa8-b068-d90c4e35f5dc",
		Action:       " http.post ",
		ResourceType: " http_request ",
		Limit:        500,
	}
	q.Normalize()
	if q.TenantID != "tenant-1" || q.Action != "http.post" || q.ResourceType != "http_request" || q.Limit != 50 {
		t.Fatalf("normalized query = %#v", q)
	}
	if err := q.Validate(); err != nil {
		t.Fatalf("expected valid query: %v", err)
	}
}

func TestAuditLogQueryRejectsInvalidActorUserID(t *testing.T) {
	q := AuditLogQuery{TenantID: "tenant-1", ActorUserID: "bad-user-id", Limit: 10}
	q.Normalize()
	if err := q.Validate(); err == nil {
		t.Fatal("expected actor_user_id validation error")
	}
}

func TestAuditLogQueryRejectsInvalidTimeRange(t *testing.T) {
	q := AuditLogQuery{
		TenantID: "tenant-1",
		From:     time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC),
		Limit:    10,
	}
	q.Normalize()
	if err := q.Validate(); err == nil {
		t.Fatal("expected time range validation error")
	}
}

func TestParseAuditLogTime(t *testing.T) {
	cases := []string{
		"2026-05-28T10:11:12Z",
		"2026-05-28T10:11:12.123456789Z",
		"2026-05-28",
	}
	for _, tc := range cases {
		if _, err := ParseAuditLogTime(tc); err != nil {
			t.Fatalf("parse %q: %v", tc, err)
		}
	}
	if _, err := ParseAuditLogTime("2026/05/28"); err == nil {
		t.Fatal("expected invalid time format error")
	}
}

func TestAuditLogCursorEncodeDecode(t *testing.T) {
	item := AuditLog{
		ID:        "7b7d45a7-10f7-4aa8-b068-d90c4e35f5dc",
		CreatedAt: time.Date(2026, 5, 28, 10, 11, 12, 123456789, time.UTC),
	}
	raw := EncodeAuditLogCursor(item)
	if raw == "" {
		t.Fatal("expected cursor")
	}
	cursor, err := DecodeAuditLogCursor(raw)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if cursor.ID != item.ID || !cursor.CreatedAt.Equal(item.CreatedAt) {
		t.Fatalf("decoded cursor = %#v", cursor)
	}
}

func TestAuditLogCursorDecodeRejectsInvalidCursor(t *testing.T) {
	for _, raw := range []string{"bad", "bm90LWVub3VnaA", "MjAyNi0wNS0yOFQxMDoxMToxMlo=|bad"} {
		if _, err := DecodeAuditLogCursor(raw); err == nil {
			t.Fatalf("expected invalid cursor error for %q", raw)
		}
	}
}
