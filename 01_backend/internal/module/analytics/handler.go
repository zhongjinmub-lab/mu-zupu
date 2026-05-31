package analytics

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/tenant"
	"mu-agent-saas/pkg/response"
)

type Handler struct {
	Repo Repository
}

func NewHandler(repo Repository) Handler {
	return Handler{Repo: repo}
}

func (h Handler) Summary(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.Summary(c.Request.Context(), t.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50090, err.Error())
		return
	}
	response.OK(c, item)
}

func (h Handler) ExportSummary(c *gin.Context) {
	t, ok := tenant.CurrentTenant(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, 40010, "tenant context is required")
		return
	}
	item, err := h.Repo.Summary(c.Request.Context(), t.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 50090, err.Error())
		return
	}
	filename := "analytics-summary-" + time.Now().Format("20060102-150405") + ".csv"
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"section", "name", "status", "quantity", "unit", "occurred_at", "generated_at"})
	writeMetricRow(writer, "resource", "知识库", "", item.Resource.KnowledgeBases, "个", item.GeneratedAt)
	writeMetricRow(writer, "resource", "文件", "", item.Resource.Files, "个", item.GeneratedAt)
	writeMetricRow(writer, "resource", "文档", "", item.Resource.Documents, "篇", item.GeneratedAt)
	writeMetricRow(writer, "resource", "文档 Chunk", "", item.Resource.DocumentChunks, "段", item.GeneratedAt)
	writeMetricRow(writer, "resource", "待向量化 Chunk", "", item.Resource.PendingChunks, "段", item.GeneratedAt)
	writeMetricRow(writer, "resource", "文档任务", "", item.Resource.DocumentJobs, "个", item.GeneratedAt)
	writeMetricRow(writer, "resource", "智能体", "", item.Resource.Agents, "个", item.GeneratedAt)
	writeMetricRow(writer, "resource", "会话", "", item.Resource.Conversations, "个", item.GeneratedAt)
	writeMetricRow(writer, "resource", "消息", "", item.Resource.Messages, "条", item.GeneratedAt)
	writeMetricRow(writer, "resource", "成员", "", item.Resource.Members, "人", item.GeneratedAt)
	writeMetricRow(writer, "resource", "邀请", "", item.Resource.Invitations, "个", item.GeneratedAt)
	writeFloatRow(writer, "business", "已支付收入", "", item.Business.TotalRevenueCNY, "CNY", item.GeneratedAt)
	writeMetricRow(writer, "business", "已支付订单", "", item.Business.PaidOrders, "单", item.GeneratedAt)
	writeMetricRow(writer, "business", "待支付订单", "", item.Business.PendingOrders, "单", item.GeneratedAt)
	writeMetricRow(writer, "business", "异常支付单", "", item.Business.FailedPayments, "单", item.GeneratedAt)
	for _, row := range item.Business.Orders {
		writeMetricRow(writer, "business_order", "订单状态", row.Status, row.Count, "单", item.GeneratedAt)
	}
	for _, row := range item.Business.Payments {
		writeMetricRow(writer, "business_payment", "支付状态", row.Status, row.Count, "单", item.GeneratedAt)
	}
	for _, row := range item.Business.Licenses {
		writeMetricRow(writer, "business_license", "授权状态", row.Status, row.Count, "个", item.GeneratedAt)
	}
	writeMetricRow(writer, "genealogy", "智能体节点", "", item.Genealogy.Nodes, "个", item.GeneratedAt)
	writeMetricRow(writer, "genealogy", "族谱关系", "", item.Genealogy.Edges, "条", item.GeneratedAt)
	writeMetricRow(writer, "genealogy", "根节点", "", item.Genealogy.Roots, "个", item.GeneratedAt)
	writeMetricRow(writer, "genealogy", "孤立节点", "", item.Genealogy.Isolated, "个", item.GeneratedAt)
	for _, row := range item.Genealogy.RelationTypes {
		writeMetricRow(writer, "genealogy_relation", "关系类型", row.Status, row.Count, "条", item.GeneratedAt)
	}
	for _, row := range item.UsageTrend {
		_ = writer.Write([]string{
			"usage_trend",
			row.Metric,
			"",
			strconv.FormatFloat(row.Quantity, 'f', -1, 64),
			row.Unit,
			row.Date,
			item.GeneratedAt.Format(time.RFC3339),
		})
	}
	for _, row := range item.Risks {
		writeMetricRow(writer, "risk", row.Message, row.Level+" "+row.Code, row.Count, "项", item.GeneratedAt)
	}
	for _, row := range item.RecentActions {
		_ = writer.Write([]string{
			"recent_action",
			row.Action,
			row.ResourceType + " " + row.ActorUserID,
			"1",
			"次",
			row.CreatedAt.Format(time.RFC3339),
			item.GeneratedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
}

func writeMetricRow(writer *csv.Writer, section, name, status string, quantity int64, unit string, generatedAt time.Time) {
	_ = writer.Write([]string{section, name, status, strconv.FormatInt(quantity, 10), unit, "", generatedAt.Format(time.RFC3339)})
}

func writeFloatRow(writer *csv.Writer, section, name, status string, quantity float64, unit string, generatedAt time.Time) {
	_ = writer.Write([]string{section, name, status, strconv.FormatFloat(quantity, 'f', -1, 64), unit, "", generatedAt.Format(time.RFC3339)})
}
