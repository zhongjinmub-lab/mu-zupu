package file

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/auth"
	"mu-agent-saas/internal/module/billing"
	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/pkg/response"
	"mu-agent-saas/pkg/storage"
)

type Handler struct {
	Repo           Repository
	Storage        storage.Client
	MaxUploadBytes int64
	Usage          billing.Repository
}

func NewHandler(repo Repository, storageClient storage.Client, maxUploadBytes int64, usage billing.Repository) Handler {
	return Handler{Repo: repo, Storage: storageClient, MaxUploadBytes: maxUploadBytes, Usage: usage}
}

func (h Handler) Upload(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 40101, "not authenticated")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.MaxUploadBytes)
	header, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40030, "file is required")
		return
	}
	if header.Size <= 0 {
		response.Error(c, http.StatusBadRequest, 40031, "file is empty")
		return
	}
	if header.Size > h.MaxUploadBytes {
		response.Error(c, http.StatusRequestEntityTooLarge, 41301, "file is too large")
		return
	}
	if err := h.Usage.EnsureQuota(c.Request.Context(), t.ID, billing.MetricFileUploadBytes, float64(header.Size)); err != nil {
		writeQuotaError(c, err)
		return
	}
	src, err := header.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40032, err.Error())
		return
	}
	defer src.Close()

	fileID := h.Repo.NewID()
	filename := sanitizeFilename(header.Filename)
	key := objectKey(t.ID, fileID, filename)
	contentType := header.Header.Get("Content-Type")
	hash := sha256.New()
	reader := io.TeeReader(src, hash)

	if err := h.Storage.EnsureBucket(c.Request.Context()); err != nil {
		response.Error(c, http.StatusInternalServerError, 50030, err.Error())
		return
	}
	if err := h.Storage.Put(c.Request.Context(), key, reader, header.Size, contentType); err != nil {
		response.Error(c, http.StatusInternalServerError, 50031, err.Error())
		return
	}

	item, err := h.Repo.Create(c.Request.Context(), CreateFileInput{
		TenantID:  t.ID,
		Bucket:    h.Storage.Bucket,
		ObjectKey: key,
		Filename:  filename,
		MimeType:  contentType,
		SizeBytes: header.Size,
		Checksum:  hex.EncodeToString(hash.Sum(nil)),
		CreatedBy: user.ID,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50032, err.Error())
		return
	}
	item.PublicURL = publicURL(h.Storage.PublicBase, item.ObjectKey)
	_ = h.Usage.Record(c.Request.Context(), billing.RecordUsageInput{
		TenantID:    t.ID,
		SubjectType: "file",
		SubjectID:   item.ID,
		Metric:      billing.MetricFileUploadBytes,
		Quantity:    float64(item.SizeBytes),
		Unit:        "bytes",
		RequestID:   c.GetString("request_id"),
		Metadata: map[string]any{
			"filename": item.Filename,
			"mime":     item.MimeType,
		},
	})
	response.OK(c, item)
}

func (h Handler) List(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.Repo.List(c.Request.Context(), t.ID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50033, err.Error())
		return
	}
	for i := range items {
		items[i].PublicURL = publicURL(h.Storage.PublicBase, items[i].ObjectKey)
	}
	response.OK(c, gin.H{"items": items})
}

func (h Handler) Download(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.Get(c.Request.Context(), t.ID, c.Param("file_id"))
	if err != nil {
		if IsNotFound(err) {
			response.Error(c, http.StatusNotFound, 40430, "file not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, 50033, err.Error())
		return
	}
	obj, err := h.Storage.Get(c.Request.Context(), item.ObjectKey)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50035, err.Error())
		return
	}
	defer obj.Close()

	contentType := strings.TrimSpace(item.MimeType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(item.SizeBytes, 10))
	c.Header("Content-Disposition", `attachment; filename="`+downloadFilename(item.Filename)+`"`)
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, obj)
}

func writeUploadError(c *gin.Context, err error) {
	if errors.Is(err, http.ErrMissingFile) {
		response.Error(c, http.StatusBadRequest, 40030, "file is required")
		return
	}
	response.Error(c, http.StatusBadRequest, 40032, err.Error())
}

func writeQuotaError(c *gin.Context, err error) {
	if check, ok := billing.IsQuotaExceeded(err); ok {
		response.Error(c, http.StatusPaymentRequired, 40201, check.Error())
		return
	}
	response.Error(c, http.StatusInternalServerError, 50060, err.Error())
}

func publicURL(base, key string) string {
	if base == "" {
		return ""
	}
	for len(base) > 0 && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	return base + "/" + key
}

func downloadFilename(name string) string {
	name = sanitizeFilename(name)
	name = strings.ReplaceAll(name, `"`, "_")
	name = strings.ReplaceAll(name, "\r", "_")
	name = strings.ReplaceAll(name, "\n", "_")
	return name
}
