package scripts_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionUpgradeRollbackScriptsArePackaged(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{"backup.sh", "smoke.sh", "upgrade.sh", "rollback.sh", "restore.sh", "restore-drill.sh", "restore-config.sh"} {
		path := filepath.Join(root, "deploy", "production", "scripts", name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("missing production script %s: %v", name, err)
		}
		if info.IsDir() {
			t.Fatalf("production script should be a file: %s", path)
		}
	}

	buildScript := readFile(t, filepath.Join(root, "scripts", "build_release.sh"))
	buildScriptPS := readFile(t, filepath.Join(root, "scripts", "build_release.ps1"))
	for _, want := range []string{"upgrade.sh", "rollback.sh", "restore.sh", "restore-drill.sh", "restore-config.sh"} {
		if !strings.Contains(buildScript, want) {
			t.Fatalf("build_release.sh should verify %s is packaged", want)
		}
		if !strings.Contains(buildScriptPS, want) {
			t.Fatalf("build_release.ps1 should verify %s is packaged", want)
		}
	}
	for _, want := range []string{"frontend/index.html", "frontend/assets/app.js", "frontend/assets/app.css"} {
		if !strings.Contains(buildScript, want) {
			t.Fatalf("build_release.sh should verify %s is packaged", want)
		}
	}
	for _, want := range []string{"index.html", "assets/app.js", "assets/app.css"} {
		if !strings.Contains(buildScriptPS, want) {
			t.Fatalf("build_release.ps1 should verify frontend file %s is packaged", want)
		}
	}
}

func TestProductionUpgradeRollbackDocsMentionSafeSteps(t *testing.T) {
	root := repoRoot(t)
	readme := readFile(t, filepath.Join(root, "deploy", "production", "README.md"))
	backendReadme := readFile(t, filepath.Join(root, "README.md"))
	for _, want := range []string{"升级", "回滚", "备份恢复演练", "backup.sh", "smoke.sh", "restore-drill.sh", "restore-config.sh", "MIGRATION_STEPS"} {
		if !strings.Contains(readme, want) {
			t.Fatalf("production README should mention %s", want)
		}
	}
	for _, want := range []string{"restore-config.sh", "公网 API", "前端静态资源", "Nginx 配置"} {
		if !strings.Contains(backendReadme, want) {
			t.Fatalf("backend README should mention %s", want)
		}
	}

	upgrade := readFile(t, filepath.Join(root, "deploy", "production", "scripts", "upgrade.sh"))
	for _, want := range []string{"backup.sh", "mu-agent-migrate up", "smoke.sh", "rollback/last_release_path", "frontend"} {
		if !strings.Contains(upgrade, want) {
			t.Fatalf("upgrade.sh should include %s", want)
		}
	}

	rollback := readFile(t, filepath.Join(root, "deploy", "production", "scripts", "rollback.sh"))
	for _, want := range []string{"backup.sh", "MIGRATION_STEPS", "mu-agent-migrate down", "smoke.sh", "frontend"} {
		if !strings.Contains(rollback, want) {
			t.Fatalf("rollback.sh should include %s", want)
		}
	}

	restore := readFile(t, filepath.Join(root, "deploy", "production", "scripts", "restore.sh"))
	for _, want := range []string{"CONFIRM_RESTORE=yes", "dropdb", "createdb", "gzip -dc", "smoke.sh"} {
		if !strings.Contains(restore, want) {
			t.Fatalf("restore.sh should include %s", want)
		}
	}

	drill := readFile(t, filepath.Join(root, "deploy", "production", "scripts", "restore-drill.sh"))
	for _, want := range []string{"mu_agent_saas_restore_drill", "schema_migrations", "REQUIRED_TABLES", "webhook_deliveries", "agent_genealogy", "KEEP_DRILL_DB", "dropdb", "restore drill ok"} {
		if !strings.Contains(drill, want) {
			t.Fatalf("restore-drill.sh should include %s", want)
		}
	}

	restoreConfig := readFile(t, filepath.Join(root, "deploy", "production", "scripts", "restore-config.sh"))
	for _, want := range []string{"CONFIRM_CONFIG_RESTORE=yes", "mu_agent_saas_config_", "frontend", "nginx", "systemd", "smoke.sh"} {
		if !strings.Contains(restoreConfig, want) {
			t.Fatalf("restore-config.sh should include %s", want)
		}
	}
}

func TestProductionBackupIncludesRuntimeAndFrontendFiles(t *testing.T) {
	root := repoRoot(t)
	backup := readFile(t, filepath.Join(root, "deploy", "production", "scripts", "backup.sh"))
	for _, want := range []string{
		"mu-agent-saas/docker-compose.yml",
		"mu-agent-saas/mu-agent-saas.env",
		"mu-agent-saas/migrations",
		"mu-agent-saas/scripts",
		"mu-agent-saas/frontend",
		"mu-agent-saas/nginx",
		"mu-agent-saas/systemd",
	} {
		if !strings.Contains(backup, want) {
			t.Fatalf("backup.sh should include %s in config archive", want)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, ".."))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
