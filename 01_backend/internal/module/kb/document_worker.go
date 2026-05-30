package kb

import (
	"context"
	"errors"
	"io"
	"strings"

	filemodule "mu-agent-saas/internal/module/file"
	"mu-agent-saas/pkg/storage"
)

type DocumentJobRunner struct {
	KB      Repository
	Files   filemodule.Repository
	Storage storage.Client
}

func NewDocumentJobRunner(kbRepo Repository, fileRepo filemodule.Repository, storageClient storage.Client) DocumentJobRunner {
	return DocumentJobRunner{KB: kbRepo, Files: fileRepo, Storage: storageClient}
}

func (r DocumentJobRunner) Run(ctx context.Context, jobs []DocumentJob) RunDocumentJobsResponse {
	var out RunDocumentJobsResponse
	for _, job := range jobs {
		documentID, err := r.Process(ctx, job)
		if err != nil {
			out.Failed++
			_ = r.KB.MarkDocumentJobFailed(ctx, job.TenantID, job.KnowledgeBaseID, job.ID, err)
			continue
		}
		if err := r.KB.MarkDocumentJobSuccess(ctx, job.TenantID, job.KnowledgeBaseID, job.ID, documentID); err != nil {
			out.Failed++
			continue
		}
		out.Processed++
	}
	return out
}

func (r DocumentJobRunner) Process(ctx context.Context, job DocumentJob) (string, error) {
	f, err := r.Files.Get(ctx, job.TenantID, job.FileID)
	if err != nil {
		return "", err
	}
	if !isPlainTextFile(f.MimeType, f.Filename) {
		return "", errors.New("only plain text or markdown files are supported")
	}
	obj, err := r.Storage.Get(ctx, f.ObjectKey)
	if err != nil {
		return "", err
	}
	defer obj.Close()
	data, err := io.ReadAll(io.LimitReader(obj, 10<<20))
	if err != nil {
		return "", err
	}
	chunks := SplitTextChunks(string(data), job.MaxChars, job.OverlapChars)
	if job.JobType == "rebuild" {
		if job.DocumentID == "" {
			return "", errors.New("document_id is required for rebuild job")
		}
		doc, err := r.KB.RebuildDocumentChunks(ctx, job.TenantID, job.KnowledgeBaseID, job.DocumentID, chunks)
		if err != nil {
			return "", err
		}
		return doc.ID, nil
	}
	title := strings.TrimSpace(job.Title)
	if title == "" {
		title = strings.TrimSpace(f.Filename)
	}
	doc, err := r.KB.CreateDocumentFromFile(ctx, job.TenantID, job.KnowledgeBaseID, f.ID, CreateDocumentRequest{
		Title:         title,
		SourceType:    "file",
		SourceURI:     f.ObjectKey,
		MimeType:      f.MimeType,
		ContentSHA256: f.Checksum,
		Metadata:      map[string]any{"document_job_id": job.ID},
	}, chunks)
	if err != nil {
		return "", err
	}
	return doc.ID, nil
}
