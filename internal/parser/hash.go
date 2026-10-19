package parser

import (
	"crypto/sha256"
	"encoding/hex"
)

// ComputeBlockHash returns the SHA256 hex digest of rawContent.
// rawContent is the exact string accumulated between the header line and the @@end line:
// all intermediate lines joined with "\n", no leading or trailing "\n".
func ComputeBlockHash(rawContent string) string {
	h := sha256.Sum256([]byte(rawContent))
	return hex.EncodeToString(h[:])
}

// VerifyBlockHash returns true if the SHA256 of rawContent matches declaredHash.
func VerifyBlockHash(rawContent, declaredHash string) bool {
	return ComputeBlockHash(rawContent) == declaredHash
}
