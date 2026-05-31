package tenant

import "testing"

func TestDefaultRolePermissionMatrixCoversAllRoles(t *testing.T) {
	matrix := DefaultRolePermissionMatrix()
	for _, role := range DefaultRoles() {
		if _, ok := matrix.Matrix[role]; !ok {
			t.Fatalf("role %q missing from matrix", role)
		}
	}
}

func TestOwnerHasAllPermissions(t *testing.T) {
	matrix := DefaultRolePermissionMatrix()
	for _, perm := range DefaultPermissions() {
		found := false
		for _, p := range matrix.Matrix["owner"] {
			if p == perm.Code {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("owner should have permission %q", perm.Code)
		}
	}
}

func TestViewerCannotManage(t *testing.T) {
	managePerms := []string{"agent.create", "kb.manage", "tenant.members", "billing.manage"}
	for _, code := range managePerms {
		if HasPermission("viewer", code) {
			t.Fatalf("viewer should not have %q", code)
		}
	}
}

func TestViewerCanView(t *testing.T) {
	viewPerms := []string{"agent.chat", "kb.search", "workflow.view", "channel.view", "billing.view", "audit.view"}
	for _, code := range viewPerms {
		if !HasPermission("viewer", code) {
			t.Fatalf("viewer should have %q", code)
		}
	}
}

func TestMemberCannotManageTenant(t *testing.T) {
	if HasPermission("member", "tenant.members") {
		t.Fatal("member should not manage tenant members")
	}
	if HasPermission("member", "billing.manage") {
		t.Fatal("member should not manage billing")
	}
}

func TestMemberCanManageAgent(t *testing.T) {
	if !HasPermission("member", "agent.create") {
		t.Fatal("member should be able to create agents")
	}
	if !HasPermission("member", "workflow.manage") {
		t.Fatal("member should be able to manage workflows")
	}
}

func TestHasPermissionUnknownRole(t *testing.T) {
	if HasPermission("unknown_role", "agent.create") {
		t.Fatal("unknown role should have no permissions")
	}
}
