package context

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rishirajsingh/wfl/pkg/types"
)

func writeFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestResolveContextAnnotation_Basic(t *testing.T) {
	dir := t.TempDir()
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = "line content"
	}
	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	writeFixture(t, dir, "src.go", content)

	ann := types.Annotation{Type: types.AnnotationContext, Value: "src.go:15"}
	result := ResolveContextAnnotation(ann, dir)
	if !result.Exists {
		t.Fatal("expected file to exist")
	}
	if result.LineNumber != 15 {
		t.Errorf("expected lineNumber 15, got %d", result.LineNumber)
	}
	if len(result.Excerpt) == 0 {
		t.Error("expected non-empty excerpt")
	}
}

func TestResolveContextAnnotation_NonExistent(t *testing.T) {
	ann := types.Annotation{Type: types.AnnotationContext, Value: "nonexistent.go:5"}
	result := ResolveContextAnnotation(ann, "/tmp/doesnotexist")
	if result.Exists {
		t.Error("expected Exists=false for missing file")
	}
}

func TestResolveContextAnnotation_ClipsAtStart(t *testing.T) {
	dir := t.TempDir()
	content := ""
	for i := 0; i < 10; i++ {
		content += "line\n"
	}
	writeFixture(t, dir, "tiny.go", content)

	ann := types.Annotation{Type: types.AnnotationContext, Value: "tiny.go:1"}
	result := ResolveContextAnnotation(ann, dir)
	if !result.Exists {
		t.Fatal("expected file to exist")
	}
	// Should not panic or return negative slice
	if len(result.Excerpt) == 0 {
		t.Error("expected non-empty excerpt")
	}
}

func TestResolveContextAnnotation_ClipsAtEnd(t *testing.T) {
	dir := t.TempDir()
	content := ""
	for i := 0; i < 5; i++ {
		content += "line\n"
	}
	writeFixture(t, dir, "small.go", content)

	ann := types.Annotation{Type: types.AnnotationContext, Value: "small.go:5"}
	result := ResolveContextAnnotation(ann, dir)
	if !result.Exists {
		t.Fatal("expected file to exist")
	}
	if len(result.Excerpt) == 0 {
		t.Error("expected non-empty excerpt")
	}
}

func TestResolveAll_Metadata(t *testing.T) {
	dir := t.TempDir()
	block := types.ValidatedBlock{}
	block.Annotations = []types.Annotation{
		{Type: types.AnnotationAPI, Value: "internal/auth"},
		{Type: types.AnnotationDB, Value: "users.credentials"},
	}

	_, metadata := ResolveAll(block, dir)
	if len(metadata["api"]) != 1 || metadata["api"][0] != "internal/auth" {
		t.Errorf("expected api metadata, got %v", metadata["api"])
	}
	if len(metadata["db"]) != 1 || metadata["db"][0] != "users.credentials" {
		t.Errorf("expected db metadata, got %v", metadata["db"])
	}
}
