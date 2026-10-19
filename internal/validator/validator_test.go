package validator

import (
	"strings"
	"testing"

	"github.com/rishirajsingh/wfl/internal/parser"
	"github.com/rishirajsingh/wfl/pkg/types"
)

// buildWfl builds a minimal WflFile with correct hash for the intent block.
func buildWfl(t *testing.T, intentVersion, implVersion string) *types.WflFile {
	t.Helper()
	intentContent := "  actor -> svc: do()"
	hash := parser.ComputeBlockHash(intentContent)

	raw := "@@intent v" + intentVersion + " flow:TestFlow\n" +
		intentContent + "\n" +
		"@@end:sha256:" + hash + "\n\n" +
		"@@impl v" + implVersion + " flow:TestFlow\n" +
		"  # impl notes\n" +
		"@@end\n\n" +
		"@@changelog flow:TestFlow\n" +
		"  v" + intentVersion + " feat: initial\n" +
		"@@end"

	wf, err := parser.ParseString(raw, "test.wfl")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	return wf
}

func TestValidate_HashMatch(t *testing.T) {
	wf := buildWfl(t, "1.0.0", "1.0.0")
	result := Validate(wf)
	if result.HasHardErrors {
		t.Errorf("expected no hard errors, got: %v", result.Errors)
	}
	if !result.Blocks[0].HashVerified {
		t.Error("expected HashVerified=true on intent block")
	}
}

func TestValidate_HashMismatch(t *testing.T) {
	wf := buildWfl(t, "1.0.0", "1.0.0")
	// Tamper the intent block's declared hash.
	wf.Blocks[0].DeclaredHash = strings.Repeat("a", 64)
	result := Validate(wf)
	if !result.HasHardErrors {
		t.Error("expected hard error for hash mismatch")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == "HASH_MISMATCH" {
			found = true
		}
	}
	if !found {
		t.Error("expected HASH_MISMATCH error code")
	}
}

func TestValidate_MissingHash(t *testing.T) {
	raw := "@@intent v1.0.0 flow:NoHash\n  actor -> svc: do()\n@@end\n\n@@impl v1.0.0 flow:NoHash\n@@end\n\n@@changelog flow:NoHash\n  v1.0.0 feat: initial\n@@end"
	wf, err := parser.ParseString(raw, "test.wfl")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	result := Validate(wf)
	if !result.HasHardErrors {
		t.Error("expected hard error for missing hash")
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == "MISSING_HASH" {
			found = true
		}
	}
	if !found {
		t.Error("expected MISSING_HASH error code")
	}
}

func TestValidate_StaleDetection(t *testing.T) {
	// intent v2.0.0, impl v1.0.0 → stale (breaking major bump)
	wf := buildWfl(t, "2.0.0", "1.0.0")
	result := Validate(wf)
	if result.HasHardErrors {
		t.Fatalf("unexpected hard errors: %v", result.Errors)
	}
	var implBlock *types.ValidatedBlock
	for i := range result.Blocks {
		if result.Blocks[i].Header.Kind == types.BlockKindImpl {
			implBlock = &result.Blocks[i]
		}
	}
	if implBlock == nil {
		t.Fatal("no impl block found")
	}
	if !implBlock.IsStale {
		t.Error("expected IsStale=true for intent v2.0.0 + impl v1.0.0")
	}
}

func TestValidate_NotStale_MinorBump(t *testing.T) {
	// intent v1.1.0, impl v1.0.0 → not stale (minor bump)
	wf := buildWfl(t, "1.1.0", "1.0.0")
	result := Validate(wf)
	for i := range result.Blocks {
		if result.Blocks[i].Header.Kind == types.BlockKindImpl {
			if result.Blocks[i].IsStale {
				t.Error("expected IsStale=false for minor bump")
			}
		}
	}
}

func TestGetFlowStatus_ImplMissing(t *testing.T) {
	intentContent := "  actor -> svc: do()"
	hash := parser.ComputeBlockHash(intentContent)
	raw := "@@intent v1.0.0 flow:NoImpl\n" + intentContent + "\n@@end:sha256:" + hash + "\n\n@@changelog flow:NoImpl\n  v1.0.0 feat: initial\n@@end"
	wf, _ := parser.ParseString(raw, "test.wfl")
	Validate(wf)
	status, err := GetFlowStatus(wf, "NoImpl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Drift != types.DriftImplMissing {
		t.Errorf("expected DriftImplMissing, got %q", status.Drift)
	}
}

func TestGetFlowStatus_InSync(t *testing.T) {
	wf := buildWfl(t, "1.0.0", "1.0.0")
	Validate(wf)
	status, err := GetFlowStatus(wf, "TestFlow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Drift != types.DriftInSync {
		t.Errorf("expected DriftInSync, got %q", status.Drift)
	}
}

func TestGetFlowStatus_Stale(t *testing.T) {
	wf := buildWfl(t, "2.0.0", "1.0.0")
	Validate(wf)
	status, err := GetFlowStatus(wf, "TestFlow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Drift != types.DriftStale {
		t.Errorf("expected DriftStale, got %q", status.Drift)
	}
	if !status.IsStale {
		t.Error("expected IsStale=true")
	}
}
