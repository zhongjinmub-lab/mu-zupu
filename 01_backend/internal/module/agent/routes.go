package agent

import (
	"github.com/gin-gonic/gin"

	"mu-agent-saas/internal/module/tenant"
)

func RegisterRoutes(rg *gin.RouterGroup, h Handler, limiters ...gin.HandlerFunc) {
	rg.GET("/agent-genealogy/graph", h.GenealogyGraph)
	rg.GET("/agent-genealogy/export", h.ExportGenealogyGraph)
	rg.GET("/agents", h.ListAgents)
	rg.GET("/agents/tool-safety-policy", h.ToolSafetyPolicy)
	rg.GET("/agents/conversation-orchestration-policy", h.ConversationOrchestrationPolicy)
	rg.GET("/tools", h.ListTools)
	rg.GET("/tool-call-logs", h.ListToolCallLogs)
	rg.GET("/agents/:agent_id", h.GetAgent)
	rg.GET("/agents/:agent_id/conversations", h.ListConversations)
	rg.GET("/agents/:agent_id/conversations/:conversation_id/messages", h.ListMessages)
	rg.GET("/agents/:agent_id/knowledge-bases", h.ListKnowledgeBases)

	write := rg.Group("")
	write.Use(tenant.RequireTenantWriter())
	write.POST("/agents", h.CreateAgent)
	write.PUT("/agents/:agent_id", h.UpdateAgent)
	write.POST("/agents/:agent_id/publish", h.PublishAgent)
	write.POST("/agents/:agent_id/rollback", h.RollbackAgent)
	write.POST("/agents/:agent_id/test-chat", h.TestChat)
	write.POST("/agents/:agent_id/chat", h.Chat)
	write.POST("/tools/:tool_id/test", h.TestTool)
	stream := write.Group("")
	stream.Use(limiters...)
	stream.POST("/agents/:agent_id/chat/stream", h.ChatStream)
	write.DELETE("/agents/:agent_id", h.ArchiveAgent)
	write.POST("/agents/:agent_id/knowledge-bases", h.BindKnowledgeBase)
	write.DELETE("/agents/:agent_id/knowledge-bases/:kb_id", h.UnbindKnowledgeBase)
	write.POST("/agent-genealogy/edges", h.CreateGenealogyEdge)
	write.DELETE("/agent-genealogy/edges/:edge_id", h.DeleteGenealogyEdge)
}
