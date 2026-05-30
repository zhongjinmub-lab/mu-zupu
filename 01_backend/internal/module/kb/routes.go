package kb

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"mu-agent-saas/internal/module/tenant"
)

func RegisterRoutes(rg *gin.RouterGroup, db *pgxpool.Pool) {
	RegisterRoutesWithHandler(rg, NewHandler(db))
}

func RegisterRoutesWithHandler(rg *gin.RouterGroup, h Handler) {
	rg.GET("/kbs", h.ListKnowledgeBases)
	rg.GET("/kbs/:kb_id/documents", h.ListDocuments)
	rg.GET("/kbs/:kb_id/documents/:document_id", h.GetDocument)
	rg.GET("/kbs/:kb_id/documents/:document_id/chunks", h.ListDocumentChunks)
	rg.GET("/kbs/:kb_id/document-jobs", h.ListDocumentJobs)
	rg.GET("/kbs/:kb_id/chunks/pending", h.ListPendingChunks)
	rg.POST("/kbs/:kb_id/search", h.SearchKnowledgeBase)

	g := rg.Group("/kb")
	g.POST("/search/vector", h.VectorSearch)
	g.POST("/search/hybrid", h.HybridSearch)

	write := rg.Group("")
	write.Use(tenant.RequireTenantWriter())
	write.POST("/kbs", h.CreateKnowledgeBase)
	write.POST("/kbs/:kb_id/documents", h.CreateDocument)
	write.POST("/kbs/:kb_id/documents/from-file", h.CreateDocumentFromFile)
	write.POST("/kbs/:kb_id/documents/:document_id/rebuild", h.RebuildDocument)
	write.DELETE("/kbs/:kb_id/documents/:document_id", h.ArchiveDocument)
	write.POST("/kbs/:kb_id/document-jobs", h.EnqueueDocumentJob)
	write.POST("/kbs/:kb_id/document-jobs/run", h.RunDocumentJobs)
	write.POST("/kbs/:kb_id/chunks", h.CreateChunk)
	write.PUT("/kbs/:kb_id/chunks/:chunk_id/embedding", h.UpdateChunkEmbedding)
	write.POST("/kbs/:kb_id/embedding/run", h.RunEmbedding)
	write.POST("/kbs/:kb_id/ask", h.AskKnowledgeBase)
}
