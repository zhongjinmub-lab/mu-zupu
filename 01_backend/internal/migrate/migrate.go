package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

type Migration struct {
	Version   string
	Name      string
	UpPath    string
	DownPath  string
	UpSQL     string
	DownSQL   string
	Checksum  string
	AppliedAt *time.Time
}

func LoadDir(dir string) ([]Migration, error) {
	upFiles, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return nil, err
	}
	items := make([]Migration, 0, len(upFiles))
	for _, upPath := range upFiles {
		base := filepath.Base(upPath)
		stem := strings.TrimSuffix(base, ".up.sql")
		version, name, err := splitStem(stem)
		if err != nil {
			return nil, err
		}
		downPath := filepath.Join(dir, stem+".down.sql")
		upSQLBytes, err := os.ReadFile(upPath)
		if err != nil {
			return nil, err
		}
		downSQLBytes, err := os.ReadFile(downPath)
		if err != nil {
			return nil, fmt.Errorf("missing down migration for %s: %w", base, err)
		}
		sum := sha256.Sum256(upSQLBytes)
		items = append(items, Migration{
			Version:  version,
			Name:     name,
			UpPath:   upPath,
			DownPath: downPath,
			UpSQL:    string(upSQLBytes),
			DownSQL:  string(downSQLBytes),
			Checksum: hex.EncodeToString(sum[:]),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Version < items[j].Version })
	return items, nil
}

func EnsureSchema(ctx context.Context, db *pgxpool.Pool) error {
	const q = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    checksum TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
	_, err := db.Exec(ctx, q)
	return err
}

func Status(ctx context.Context, db *pgxpool.Pool, dir string) ([]Migration, error) {
	items, err := LoadDir(dir)
	if err != nil {
		return nil, err
	}
	if err := EnsureSchema(ctx, db); err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT version, applied_at FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applied := map[string]time.Time{}
	for rows.Next() {
		var version string
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, err
		}
		applied[version] = appliedAt
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range items {
		if t, ok := applied[items[i].Version]; ok {
			items[i].AppliedAt = &t
		}
	}
	return items, nil
}

func Up(ctx context.Context, db *pgxpool.Pool, dir string) ([]Migration, error) {
	items, err := Status(ctx, db, dir)
	if err != nil {
		return nil, err
	}
	applied := make([]Migration, 0)
	for _, item := range items {
		if item.AppliedAt != nil {
			continue
		}
		if err := applyOne(ctx, db, item, DirectionUp); err != nil {
			return applied, err
		}
		applied = append(applied, item)
	}
	return applied, nil
}

func Down(ctx context.Context, db *pgxpool.Pool, dir string) (Migration, error) {
	items, err := Status(ctx, db, dir)
	if err != nil {
		return Migration{}, err
	}
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].AppliedAt == nil {
			continue
		}
		err := applyOne(ctx, db, items[i], DirectionDown)
		return items[i], err
	}
	return Migration{}, errors.New("no applied migration to rollback")
}

func applyOne(ctx context.Context, db *pgxpool.Pool, item Migration, direction Direction) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	switch direction {
	case DirectionUp:
		if _, err := tx.Exec(ctx, item.UpSQL); err != nil {
			return fmt.Errorf("apply %s up failed: %w", item.Version, err)
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO schema_migrations(version, name, checksum)
VALUES ($1, $2, $3)
ON CONFLICT (version) DO UPDATE SET name = EXCLUDED.name, checksum = EXCLUDED.checksum, applied_at = now()`,
			item.Version, item.Name, item.Checksum,
		); err != nil {
			return err
		}
	case DirectionDown:
		if _, err := tx.Exec(ctx, item.DownSQL); err != nil {
			return fmt.Errorf("apply %s down failed: %w", item.Version, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, item.Version); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported direction %q", direction)
	}
	return tx.Commit(ctx)
}

func splitStem(stem string) (string, string, error) {
	parts := strings.SplitN(stem, "_", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid migration filename %q", stem)
	}
	return parts[0], parts[1], nil
}

func IsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
