package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDirSortsAndRequiresDownMigration(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "000002_second.up.sql"), "SELECT 2;")
	writeFile(t, filepath.Join(dir, "000002_second.down.sql"), "SELECT -2;")
	writeFile(t, filepath.Join(dir, "000001_first.up.sql"), "SELECT 1;")
	writeFile(t, filepath.Join(dir, "000001_first.down.sql"), "SELECT -1;")

	items, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d", len(items))
	}
	if items[0].Version != "000001" || items[1].Version != "000002" {
		t.Fatalf("items not sorted: %#v", items)
	}
	if items[0].Checksum == "" {
		t.Fatal("checksum is empty")
	}
}

func TestLoadDirFailsWhenDownMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "000001_first.up.sql"), "SELECT 1;")
	if _, err := LoadDir(dir); err == nil {
		t.Fatal("expected missing down migration error")
	}
}

func TestSplitStem(t *testing.T) {
	version, name, err := splitStem("000004_auth_tenant_kb_mvp")
	if err != nil {
		t.Fatalf("splitStem error: %v", err)
	}
	if version != "000004" || name != "auth_tenant_kb_mvp" {
		t.Fatalf("version=%q name=%q", version, name)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
