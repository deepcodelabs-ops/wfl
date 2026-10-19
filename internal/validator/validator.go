package validator

import (
	"fmt"

	"github.com/rishirajsingh/wfl/internal/parser"
	"github.com/rishirajsingh/wfl/pkg/types"
)

// Validate takes a parsed WflFile and returns ValidatedBlocks with hash/stale fields
// populated, plus any hard errors found.
func Validate(wf *types.WflFile) types.ValidationResult {
	result := types.ValidationResult{
		Blocks: make([]types.ValidatedBlock, len(wf.Blocks)),
	}

	// First pass: verify hashes on all intent blocks.
	for i, vb := range wf.Blocks {
		b := vb.Block
		validated := types.ValidatedBlock{Block: b}

		if b.Header.Kind == types.BlockKindIntent {
			if b.DeclaredHash == "" {
				result.Errors = append(result.Errors, types.ValidationError{
					Block:   b,
					Code:    "MISSING_HASH",
					Message: fmt.Sprintf("@@intent block for flow %q has no hash lock", b.Header.FlowName),
				})
				result.HasHardErrors = true
				validated.HashVerified = false
			} else {
				validated.HashVerified = parser.VerifyBlockHash(b.RawContent, b.DeclaredHash)
				if !validated.HashVerified {
					result.Errors = append(result.Errors, types.ValidationError{
						Block:   b,
						Code:    "HASH_MISMATCH",
						Message: fmt.Sprintf("@@intent block for flow %q has been tampered — hash mismatch", b.Header.FlowName),
					})
					result.HasHardErrors = true
				}
			}
		} else {
			validated.HashVerified = true // impl/changelog: no hash required
		}

		result.Blocks[i] = validated
	}

	// Second pass: stale detection — pair intent + impl blocks by flowName.
	intentVersions := map[string]string{}
	for _, vb := range result.Blocks {
		if vb.Header.Kind == types.BlockKindIntent {
			intentVersions[vb.Header.FlowName] = vb.Header.Version
		}
	}

	for i, vb := range result.Blocks {
		if vb.Header.Kind != types.BlockKindImpl {
			continue
		}
		intentVer, ok := intentVersions[vb.Header.FlowName]
		if !ok {
			continue
		}
		breaking, err := IsBreakingBump(intentVer, vb.Header.Version)
		if err == nil && breaking {
			result.Blocks[i].IsStale = true
		}
	}

	// Copy validated stale flags back to the source WflFile so callers see updated state.
	wf.Blocks = result.Blocks

	return result
}

// GetFlowStatus computes the FlowStatus for a named flow.
func GetFlowStatus(wf *types.WflFile, flowName string) (types.FlowStatus, error) {
	status := types.FlowStatus{FlowName: flowName}

	var intentBlock, implBlock *types.ValidatedBlock
	for i := range wf.Blocks {
		b := &wf.Blocks[i]
		if b.Header.FlowName != flowName {
			continue
		}
		switch b.Header.Kind {
		case types.BlockKindIntent:
			intentBlock = b
		case types.BlockKindImpl:
			implBlock = b
		}
	}

	if intentBlock == nil {
		return status, fmt.Errorf("flow %q not found", flowName)
	}

	status.IntentVersion = intentBlock.Header.Version
	status.HashVerified = intentBlock.HashVerified

	if !intentBlock.HashVerified {
		status.Drift = types.DriftHashMismatch
		return status, nil
	}

	if implBlock == nil {
		status.Drift = types.DriftImplMissing
		return status, nil
	}

	status.ImplVersion = implBlock.Header.Version
	status.IsStale = implBlock.IsStale

	if implBlock.IsStale {
		status.Drift = types.DriftStale
	} else {
		status.Drift = types.DriftInSync
	}

	return status, nil
}

// ListAllStatuses returns FlowStatus for every unique flow in the file.
func ListAllStatuses(wf *types.WflFile) []types.FlowStatus {
	seen := map[string]bool{}
	var statuses []types.FlowStatus
	for _, b := range wf.Blocks {
		if b.Header.Kind == types.BlockKindIntent && !seen[b.Header.FlowName] {
			seen[b.Header.FlowName] = true
			s, err := GetFlowStatus(wf, b.Header.FlowName)
			if err == nil {
				statuses = append(statuses, s)
			}
		}
	}
	return statuses
}
