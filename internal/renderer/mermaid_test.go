package renderer

import (
	"strings"
	"testing"

	"github.com/rishirajsingh/wfl/internal/parser"
	"github.com/rishirajsingh/wfl/pkg/types"
)

func makeIntentBlock(t *testing.T, content string) types.ValidatedBlock {
	t.Helper()
	hash := parser.ComputeBlockHash(content)
	raw := "@@intent v1.0.0 flow:Test\n" + content + "\n@@end:sha256:" + hash
	wf, err := parser.ParseString(raw, "test.wfl")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return wf.Blocks[0]
}

func TestRenderArrow(t *testing.T) {
	block := makeIntentBlock(t, "  client -> gateway: POST /login")
	out := RenderFlowToMermaid(block)
	if !strings.Contains(out, "client->>gateway: POST /login") {
		t.Errorf("expected Mermaid arrow, got:\n%s", out)
	}
}

func TestRenderMultipleArrows(t *testing.T) {
	content := "  a -> b: step1\n  b -> c: step2\n  c -> a: done"
	block := makeIntentBlock(t, content)
	out := RenderFlowToMermaid(block)
	if !strings.Contains(out, "a->>b: step1") {
		t.Errorf("missing first arrow in:\n%s", out)
	}
	if !strings.Contains(out, "b->>c: step2") {
		t.Errorf("missing second arrow in:\n%s", out)
	}
	if !strings.Contains(out, "c->>a: done") {
		t.Errorf("missing third arrow in:\n%s", out)
	}
}

func TestRenderContextAnnotation(t *testing.T) {
	content := "  client -> svc: call()\n    @context: src/main.go:10"
	block := makeIntentBlock(t, content)
	out := RenderFlowToMermaid(block)
	if !strings.Contains(out, "@context src/main.go:10") {
		t.Errorf("expected context note in:\n%s", out)
	}
}

func TestRenderFencedMermaidBlock(t *testing.T) {
	block := makeIntentBlock(t, "  a -> b: go")
	out := RenderFlowToMermaid(block)
	if !strings.Contains(out, "```mermaid") {
		t.Errorf("expected mermaid fence in:\n%s", out)
	}
	if !strings.Contains(out, "sequenceDiagram") {
		t.Errorf("expected sequenceDiagram in:\n%s", out)
	}
}

func TestRenderCommentAsNote(t *testing.T) {
	content := "  # this is a note\n  a -> b: action()"
	block := makeIntentBlock(t, content)
	out := RenderFlowToMermaid(block)
	if !strings.Contains(out, "Note over all: this is a note") {
		t.Errorf("expected comment as Note in:\n%s", out)
	}
}
