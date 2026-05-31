package channel

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrChannelNotFound 表示当前租户下不存在指定渠道。
var ErrChannelNotFound = errors.New("channel not found")

// Repository 提供渠道接入点的持久化访问。
type Repository struct {
	DB *pgxpool.Pool
}

// NewRepository 构造渠道 Repository。
func NewRepository(db *pgxpool.Pool) Repository {
	return Repository{DB: db}
}

const channelColumns = `id::text, tenant_id::text, agent_id::text, type, name, status, channel_key, config, COALESCE(created_by::text, ''), created_at, updated_at`

// Create 创建渠道接入点，channel_key 由数据库默认值生成。
func (r Repository) Create(ctx context.Context, tenantID, userID string, req CreateChannelRequest) (Channel, error) {
	config, err := marshalConfig(req.Config)
	if err != nil {
		return Channel{}, err
	}
	const q = `
INSERT INTO agent_channels(tenant_id, agent_id, type, name, config, created_by)
VALUES ($1, $2::uuid, $3, $4, $5::jsonb, NULLIF($6, '')::uuid)
RETURNING ` + channelColumns
	rows, err := r.DB.Query(ctx, q, tenantID, req.AgentID, req.Type, req.Name, config, userID)
	if err != nil {
		return Channel{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return Channel{}, rows.Err()
		}
		return Channel{}, ErrChannelNotFound
	}
	return scanChannel(rows)
}

// List 返回当前租户未归档的渠道列表。
func (r Repository) List(ctx context.Context, tenantID string) ([]Channel, error) {
	const q = `
SELECT ` + channelColumns + `
FROM agent_channels
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC`
	rows, err := r.DB.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Channel, 0)
	for rows.Next() {
		item, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// Get 返回指定渠道（按当前租户隔离）。
func (r Repository) Get(ctx context.Context, tenantID, channelID string) (Channel, error) {
	const q = `
SELECT ` + channelColumns + `
FROM agent_channels
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	rows, err := r.DB.Query(ctx, q, channelID, tenantID)
	if err != nil {
		return Channel{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return Channel{}, rows.Err()
		}
		return Channel{}, ErrChannelNotFound
	}
	return scanChannel(rows)
}

// SetStatus 设置渠道状态（启用/禁用）。
func (r Repository) SetStatus(ctx context.Context, tenantID, channelID, status string) (Channel, error) {
	const q = `
UPDATE agent_channels
SET status = $3, updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
RETURNING ` + channelColumns
	rows, err := r.DB.Query(ctx, q, channelID, tenantID, status)
	if err != nil {
		return Channel{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Channel{}, ErrChannelNotFound
	}
	return scanChannel(rows)
}

// Archive 归档（软删除）渠道。
func (r Repository) Archive(ctx context.Context, tenantID, channelID string) error {
	const q = `
UPDATE agent_channels
SET status = 'archived', deleted_at = now(), updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	tag, err := r.DB.Exec(ctx, q, channelID, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrChannelNotFound
	}
	return nil
}

// marshalConfig 将配置序列化为 JSON 字符串，nil 时返回空对象。
func marshalConfig(config map[string]any) (string, error) {
	if config == nil {
		config = map[string]any{}
	}
	b, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// scanChannel 扫描渠道记录，config 列以原始 JSON 字节解码。
func scanChannel(rows pgx.Rows) (Channel, error) {
	var item Channel
	var config []byte
	err := rows.Scan(
		&item.ID, &item.TenantID, &item.AgentID, &item.Type, &item.Name, &item.Status,
		&item.ChannelKey, &config, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return Channel{}, err
	}
	if len(config) > 0 {
		if err := json.Unmarshal(config, &item.Config); err != nil {
			return Channel{}, err
		}
	}
	if item.Config == nil {
		item.Config = map[string]any{}
	}
	return item, nil
}

// GetByChannelKey 按公开接入凭据 channel_key 查询已启用渠道（不限租户，用于外部接入初始化）。
func (r Repository) GetByChannelKey(ctx context.Context, channelKey string) (Channel, error) {
	const q = `
SELECT ` + channelColumns + `
FROM agent_channels
WHERE channel_key = $1 AND status = 'enabled' AND deleted_at IS NULL`
	rows, err := r.DB.Query(ctx, q, channelKey)
	if err != nil {
		return Channel{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return Channel{}, rows.Err()
		}
		return Channel{}, ErrChannelNotFound
	}
	return scanChannel(rows)
}
