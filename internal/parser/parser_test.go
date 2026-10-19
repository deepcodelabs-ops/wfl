package parser

import (
	"strings"
	"testing"

	"github.com/rishirajsingh/wfl/pkg/types"
)

const sampleWfl = `@@intent v1.0.0 flow:UserAuth
  # feat: initial auth flow

  client -> gateway: POST /login
  gateway -> authService: validateCredentials()
    @context: src/middleware/jwt.ts:42
    @api: internal/auth
    @db: users.credentials
  authService -> tokenStore: issueJWT()
  gateway -> client: 200 + token
@@end:sha256:PLACEHOLDER

@@impl v1.0.0 flow:UserAuth
  # agent-written after implementation
@@end

@@changelog flow:UserAuth
  v1.0.0 feat: initial auth flow
@@end`

func buildSampleWithHash(t *testing.T) string {
	t.Helper()
	// Parse once without hash to get the rawContent, then compute its hash.
	noHash := strings.Replace(sampleWfl, "@@end:sha256:PLACEHOLDER", "@@end", 1)
	wf, err := ParseString(noHash, "test.wfl")
	if err != nil {
		t.Fatalf("setup parse failed: %v", err)
	}
	var intentBlock *types.Block
	for i := range wf.Blocks {
		if wf.Blocks[i].Header.Kind == types.BlockKindIntent {
			intentBlock = &wf.Blocks[i].Block
			break
		}
	}
	if intentBlock == nil {
		t.Fatal("no intent block found in setup")
	}
	hash := ComputeBlockHash(intentBlock.RawContent)
	return strings.Replace(noHash, "@@end", "@@end:sha256:"+hash, 1)
}

func TestParseString_BlockCount(t *testing.T) {
	content := buildSampleWithHash(t)
	wf, err := ParseString(content, "test.wfl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wf.Blocks) != 3 {
		t.Errorf("expected 3 blocks, got %d", len(wf.Blocks))
	}
}

func TestParseString_BlockKinds(t *testing.T) {
	content := buildSampleWithHash(t)
	wf, _ := ParseString(content, "test.wfl")
	kinds := []types.BlockKind{
		types.BlockKindIntent,
		types.BlockKindImpl,
		types.BlockKindChangelog,
	}
	for i, want := range kinds {
		if wf.Blocks[i].Header.Kind != want {
			t.Errorf("block[%d]: want kind %q, got %q", i, want, wf.Blocks[i].Header.Kind)
		}
	}
}

func TestParseString_IntentFields(t *testing.T) {
	content := buildSampleWithHash(t)
	wf, _ := ParseString(content, "test.wfl")
	b := wf.Blocks[0]
	if b.Header.Version != "1.0.0" {
		t.Errorf("want version 1.0.0, got %q", b.Header.Version)
	}
	if b.Header.FlowName != "UserAuth" {
		t.Errorf("want flowName UserAuth, got %q", b.Header.FlowName)
	}
	if b.DeclaredHash == "" {
		t.Error("expected declared hash on intent block")
	}
}

func TestParseString_Arrows(t *testing.T) {
	content := buildSampleWithHash(t)
	wf, _ := ParseString(content, "test.wfl")
	arrows := wf.Blocks[0].Arrows
	if len(arrows) != 4 {
		t.Fatalf("expected 4 arrows, got %d", len(arrows))
	}
	if arrows[0].From != "client" || arrows[0].To != "gateway" || arrows[0].Label != "POST /login" {
		t.Errorf("arrow[0] mismatch: %+v", arrows[0])
	}
}

func TestParseString_Annotations(t *testing.T) {
	content := buildSampleWithHash(t)
	wf, _ := ParseString(content, "test.wfl")
	anns := wf.Blocks[0].Annotations
	if len(anns) != 3 {
		t.Fatalf("expected 3 annotations, got %d", len(anns))
	}
	if anns[0].Type != types.AnnotationContext || anns[0].Value != "src/middleware/jwt.ts:42" {
		t.Errorf("annotation[0] mismatch: %+v", anns[0])
	}
	if anns[1].Type != types.AnnotationAPI || anns[1].Value != "internal/auth" {
		t.Errorf("annotation[1] mismatch: %+v", anns[1])
	}
}

func TestParseString_ImplNoHash(t *testing.T) {
	content := buildSampleWithHash(t)
	wf, _ := ParseString(content, "test.wfl")
	impl := wf.Blocks[1]
	if impl.DeclaredHash != "" {
		t.Errorf("expected no hash on impl block, got %q", impl.DeclaredHash)
	}
}

func TestParseString_ChangelogVersion(t *testing.T) {
	content := buildSampleWithHash(t)
	wf, _ := ParseString(content, "test.wfl")
	cl := wf.Blocks[2]
	if cl.Header.Version != "0.0.0" {
		t.Errorf("expected changelog version 0.0.0, got %q", cl.Header.Version)
	}
	if cl.Header.FlowName != "UserAuth" {
		t.Errorf("expected changelog flowName UserAuth, got %q", cl.Header.FlowName)
	}
}

func TestParseString_UnclosedBlockError(t *testing.T) {
	content := "@@intent v1.0.0 flow:TestFlow\n  actor -> svc: call()"
	_, err := ParseString(content, "test.wfl")
	if err == nil {
		t.Error("expected error for unclosed block")
	}
}

func TestParseString_RawLinesPreserved(t *testing.T) {
	content := buildSampleWithHash(t)
	wf, _ := ParseString(content, "test.wfl")
	if len(wf.RawLines) == 0 {
		t.Error("expected raw lines to be preserved")
	}
}

func TestParseString_HashRoundtrip(t *testing.T) {
	// Parse without hash, compute hash from rawContent, re-parse with hash, verify.
	noHash := `@@intent v1.0.0 flow:Simple
  actor -> svc: do()
@@end

@@impl v1.0.0 flow:Simple
@@end

@@changelog flow:Simple
  v1.0.0 feat: initial
@@end`

	wf1, err := ParseString(noHash, "test.wfl")
	if err != nil {
		t.Fatalf("first parse: %v", err)
	}
	rawContent := wf1.Blocks[0].RawContent
	hash := ComputeBlockHash(rawContent)

	withHash := strings.Replace(noHash, "@@end\n\n@@impl", "@@end:sha256:"+hash+"\n\n@@impl", 1)
	wf2, err := ParseString(withHash, "test.wfl")
	if err != nil {
		t.Fatalf("second parse: %v", err)
	}
	if wf2.Blocks[0].DeclaredHash != hash {
		t.Errorf("hash not round-tripped: want %q, got %q", hash, wf2.Blocks[0].DeclaredHash)
	}
}
