package models

// StructInfo holds information about a struct and its fields
type StructInfo struct {
	Name   string
	Fields map[string]string // field name -> field type
}

// PreloadCall represents a Preload call found in the code
type PreloadCall struct {
	File        string
	Line        int
	Relation    string
	Model       string
	LineContent string
	PreloadStr  string // The actual preload string from the line
	Scope       string // The function or scope where this preload call is found
}

// GormCall represents a GORM method call (Find, First, FirstOrCreate, etc.)
type GormCall struct {
	File        string
	Line        int
	Method      string
	LineContent string
	Scope       string
}

// VariableAssignment represents a variable assignment that might be used in GORM calls
type VariableAssignment struct {
	VarName     string
	AssignedTo  string
	Line        int
	File        string
	Scope       string
	LineContent string
}

// VariableType represents a variable with its actual Go type
type VariableType struct {
	VarName     string // The variable name (e.g., "orders", "currentInvoice")
	TypeName    string // The actual type (e.g., "[]databases.Invoice", "databases.Invoice")
	PackageName string // The package name (e.g., "databases")
	ModelName   string // The extracted model name (e.g., "Invoice")
	Scope       string // The function scope
	File        string // The file path
	Line        int    // The line number
}

// PreloadResult defines the structure for a single preload analysis result in JSON output
type PreloadResult struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Relation string `json:"relation"`
	Model    string `json:"model"`
	Variable string `json:"variable,omitempty"`
	FindLine int    `json:"find_line,omitempty"`
	Status   string `json:"status"` // "correct", "unknown", "error"
}

// AnalysisResult defines the overall structure for the JSON analysis output
type AnalysisResult struct {
	TotalPreloads int             `json:"total_preloads"`
	Correct       int             `json:"correct"`
	Unknown       int             `json:"unknown"`
	Errors        int             `json:"errors"`
	Accuracy      float64         `json:"accuracy"`
	Results       []PreloadResult `json:"results"`
}
