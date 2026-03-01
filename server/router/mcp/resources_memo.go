package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
)

// Memo resource URI scheme: memo://memos/{uid}
// Clients can read any memo they have access to by URI without calling a tool.

func (s *MCPService) registerMemoResources(mcpSrv *mcpserver.MCPServer) {
	mcpSrv.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"memo://memos/{uid}",
			"Memo",
			mcp.WithTemplateDescription("A single Memos note identified by its UID. Returns the memo content as Markdown with a YAML frontmatter header containing metadata."),
			mcp.WithTemplateMIMEType("text/markdown"),
		),
		s.handleReadMemoResource,
	)
}

func (s *MCPService) handleReadMemoResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	userID := auth.GetUserID(ctx)

	// URI format: memo://memos/{uid}
	uid := strings.TrimPrefix(req.Params.URI, "memo://memos/")
	if uid == req.Params.URI || uid == "" {
		return nil, fmt.Errorf("invalid memo URI %q: expected memo://memos/<uid>", req.Params.URI)
	}

	memo, err := s.store.GetMemo(ctx, &store.FindMemo{UID: &uid})
	if err != nil {
		return nil, fmt.Errorf("failed to get memo: %w", err)
	}
	if memo == nil {
		return nil, fmt.Errorf("memo not found: %s", uid)
	}
	if err := checkMemoAccess(memo, userID); err != nil {
		return nil, err
	}

	j := storeMemoToJSON(memo)
	text := formatMemoMarkdown(j)

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "text/markdown",
			Text:     text,
		},
	}, nil
}

// formatMemoMarkdown renders a memo as Markdown with a YAML frontmatter header.
func formatMemoMarkdown(j memoJSON) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", j.Name))
	sb.WriteString(fmt.Sprintf("creator: %s\n", j.Creator))
	sb.WriteString(fmt.Sprintf("visibility: %s\n", j.Visibility))
	sb.WriteString(fmt.Sprintf("state: %s\n", j.State))
	sb.WriteString(fmt.Sprintf("pinned: %v\n", j.Pinned))
	if len(j.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("tags: [%s]\n", strings.Join(j.Tags, ", ")))
	}
	sb.WriteString(fmt.Sprintf("create_time: %d\n", j.CreateTime))
	sb.WriteString(fmt.Sprintf("update_time: %d\n", j.UpdateTime))
	if j.Parent != "" {
		sb.WriteString(fmt.Sprintf("parent: %s\n", j.Parent))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(j.Content)

	return sb.String()
}
