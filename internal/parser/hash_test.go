package parser

import (
	"testing"
)

func TestComputeBlockHash_Deterministic(t *testing.T) {
	content := "  client -> gateway: POST /login\n  gateway -> auth: validate()"
	h1 := ComputeBlockHash(content)
	h2 := ComputeBlockHash(content)
	if h1 != h2 {
		t.Errorf("hash not deterministic: %q vs %q", h1, h2)
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars", len(h1))
	}
}

func TestComputeBlockHash_SingleCharChange(t *testing.T) {
	h1 := ComputeBlockHash("hello world")
	h2 := ComputeBlockHash("hello World")
	if h1 == h2 {
		t.Error("expected different hashes for different content")
	}
}

func TestVerifyBlockHash(t *testing.T) {
	content := "  actor -> service: action()"
	hash := ComputeBlockHash(content)
	if !VerifyBlockHash(content, hash) {
		t.Error("expected verify to return true for correct hash")
	}
	if VerifyBlockHash(content+"x", hash) {
		t.Error("expected verify to return false for tampered content")
	}
	if VerifyBlockHash(content, hash[:len(hash)-1]+"f") {
		t.Error("expected verify to return false for tampered hash")
	}
}
