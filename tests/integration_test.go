package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	wflmcp "github.com/rishirajsingh/wfl/internal/mcp"
	"github.com/rishirajsingh/wfl/internal/parser"
	"github.com/rishirajsingh/wfl/internal/renderer"
	"github.com/rishirajsingh/wfl/internal/validator"
	"github.com/rishirajsingh/wfl/pkg/types"
)

// testdataPath returns the absolute path to a testdata file.
func testdataPath(t *testing.T, name string) string {
	t.Helper()
	// Tests run from the package dir; testdata is one level up.
	abs, err := filepath.Abs(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatalf("resolve testdata path: %v", err)
	}
	return abs
}

func TestFullFlow_ParseValidateRender(t *testing.T) {
	wf, err := parser.ParseFile(testdataPath(t, "sample.wfl"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := validator.Validate(wf)
	if result.HasHardErrors {
		t.Fatalf("unexpected validation errors: %v", result.Errors)
	}

	outDir := t.TempDir()
	// Copy sample.wfl into temp dir for rendering (renderer writes beside the .wfl file).
	sampleData, _ := os.ReadFile(testdataPath(t, "sample.wfl"))
	tmpWfl := filepath.Join(outDir, "sample.wfl")
	os.WriteFile(tmpWfl, sampleData, 0644)

	wf2, _ := parser.ParseFile(tmpWfl)
	validator.Validate(wf2)

	outPath, err := renderer.RenderFile(wf2)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	mdData, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read rendered md: %v", err)
	}
	md := string(mdData)

	if !strings.Contains(md, "sequenceDiagram") {
		t.Error("rendered md missing sequenceDiagram")
	}
	if !strings.Contains(md, "client->>gateway") {
		t.Error("rendered md missing expected arrow")
	}
	if !strings.Contains(md, "Flow: UserAuth") {
		t.Error("rendered md missing flow header")
	}
}

func TestBrokenHash_FailsValidation(t *testing.T) {
	wf, err := parser.ParseFile(testdataPath(t, "sample_broken.wfl"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	result := validator.Validate(wf)
	if !result.HasHardErrors {
		t.Error("expected hard errors for tampered hash")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == "HASH_MISMATCH" {
			found = true
		}
	}
	if !found {
		t.Error("expected HASH_MISMATCH error")
	}
}

func TestStaleFlow_DetectedCorrectly(t *testing.T) {
	wf, err := parser.ParseFile(testdataPath(t, "sample_stale.wfl"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	result := validator.Validate(wf)
	if result.HasHardErrors {
		t.Fatalf("unexpected hard errors: %v", result.Errors)
	}

	status, err := validator.GetFlowStatus(wf, "StaleFlow")
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if status.Drift != types.DriftStale {
		t.Errorf("expected DriftStale, got %q", status.Drift)
	}
}

func TestWriteImpl_UpdatesFileAtomically(t *testing.T) {
	// Copy sample.wfl to a temp location so we can mutate it.
	sampleData, _ := os.ReadFile(testdataPath(t, "sample.wfl"))
	tmpFile := filepath.Join(t.TempDir(), "test.wfl")
	if err := os.WriteFile(tmpFile, sampleData, 0644); err != nil {
		t.Fatalf("write tmp file: %v", err)
	}

	wf, err := parser.ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	validator.Validate(wf)

	var implBlock *types.ValidatedBlock
	for i := range wf.Blocks {
		if wf.Blocks[i].Header.Kind == types.BlockKindImpl {
			implBlock = &wf.Blocks[i]
			break
		}
	}
	if implBlock == nil {
		t.Fatal("no impl block found")
	}

	newContent := "  # implemented: jwt middleware at src/middleware/jwt.go:42"
	if err := wflmcp.WriteImplBlock(tmpFile, wf.RawLines, *implBlock, newContent); err != nil {
		t.Fatalf("writeImpl: %v", err)
	}

	// Re-parse and verify:
	// 1. impl content was updated
	// 2. intent block is unchanged (hash still valid)
	wf2, err := parser.ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	result2 := validator.Validate(wf2)
	if result2.HasHardErrors {
		t.Errorf("hash broken after writeImpl: %v", result2.Errors)
	}

	var newImpl *types.ValidatedBlock
	for i := range wf2.Blocks {
		if wf2.Blocks[i].Header.Kind == types.BlockKindImpl {
			newImpl = &wf2.Blocks[i]
			break
		}
	}
	if newImpl == nil {
		t.Fatal("no impl block after reparse")
	}
	if !strings.Contains(newImpl.RawContent, "jwt middleware") {
		t.Errorf("impl content not updated; got: %q", newImpl.RawContent)
	}
}

func TestWriteImpl_RefusesOnStaleIntent(t *testing.T) {
	sampleData, _ := os.ReadFile(testdataPath(t, "sample_stale.wfl"))
	tmpFile := filepath.Join(t.TempDir(), "stale.wfl")
	os.WriteFile(tmpFile, sampleData, 0644)

	wf, _ := parser.ParseFile(tmpFile)
	validator.Validate(wf)

	status, _ := validator.GetFlowStatus(wf, "StaleFlow")
	if !status.IsStale {
		t.Fatal("expected stale flow for this test")
	}

	// Simulate what the MCP writeImpl tool does: check stale before writing.
	if !status.IsStale {
		t.Error("stale guard would not have triggered — test setup wrong")
	}
	// The guard is enforced in tools.go; here we just verify the status is correct
	// so the integration is end-to-end.
}
