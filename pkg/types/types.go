package types

// BlockKind discriminates the three block types in a .wfl file.
type BlockKind string

const (
	BlockKindIntent    BlockKind = "intent"
	BlockKindImpl      BlockKind = "impl"
	BlockKindChangelog BlockKind = "changelog"
)

// AnnotationType represents the supported annotation types inside intent blocks.
type AnnotationType string

const (
	AnnotationContext AnnotationType = "context"
	AnnotationAPI     AnnotationType = "api"
	AnnotationDB      AnnotationType = "db"
)

// DriftKind describes the relationship between intent and impl versions.
type DriftKind string

const (
	DriftInSync       DriftKind = "in-sync"
	DriftImplMissing  DriftKind = "impl-missing"
	DriftStale        DriftKind = "stale"
	DriftHashMismatch DriftKind = "hash-mismatch"
)

// BlockHeader contains metadata parsed from a block header line.
type BlockHeader struct {
	Kind     BlockKind
	Version  string // semver "1.0.0"; "0.0.0" for changelog blocks
	FlowName string // "UserAuth"
	RawLine  string
}

// Annotation is a single @-prefixed annotation line inside an @@intent block.
type Annotation struct {
	Type      AnnotationType
	Value     string // raw value after the colon
	BlockLine int    // 0-indexed line within the block's raw content
}

// FlowArrow is a single "from -> to: label" line inside an @@intent block.
type FlowArrow struct {
	From      string
	To        string
	Label     string
	BlockLine int
}

// Block is a fully parsed block from a .wfl file.
type Block struct {
	Header       BlockHeader
	RawContent   string       // exact bytes between header and @@end; used for hashing
	DeclaredHash string       // from @@end:sha256:...; empty for impl/changelog
	Annotations  []Annotation
	Arrows       []FlowArrow
	StartLine    int // 1-indexed line in source file
	EndLine      int
}

// ValidatedBlock extends Block with hash verification and stale status.
type ValidatedBlock struct {
	Block
	HashVerified bool
	IsStale      bool
}

// WflFile is the in-memory representation of a parsed and validated .wfl file.
type WflFile struct {
	FilePath string
	Blocks   []ValidatedBlock
	RawLines []string // original file lines, preserved for write-back
}

// FlowStatus describes the current state of a named flow.
type FlowStatus struct {
	FlowName      string
	IntentVersion string
	ImplVersion   string // empty if no @@impl block exists
	HashVerified  bool
	IsStale       bool
	Drift         DriftKind
}

// ValidationError describes a hard error found during block validation.
type ValidationError struct {
	Block   Block
	Code    string // "MISSING_HASH", "HASH_MISMATCH"
	Message string
}

// ValidationResult is the output of validating a parsed WflFile.
type ValidationResult struct {
	Blocks        []ValidatedBlock
	Errors        []ValidationError
	HasHardErrors bool
}

// ResolvedContext is the result of resolving a @context annotation.
type ResolvedContext struct {
	Annotation Annotation
	FilePath   string
	LineNumber int
	Excerpt    []string // ±20 lines around the target line
	Exists     bool
}

// Subtask is a decomposed unit of work from a flow's intent block arrows.
type Subtask struct {
	ID          string // "UserAuth-1"
	Description string // "client -> gateway: POST /login"
	From        string
	To          string
	Label       string
	Order       int // lower order runs first; same order = parallelizable
}
