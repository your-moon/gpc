package models

type PreloadResult struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Relation string `json:"relation"`
	Model    string `json:"model"`
	Status   string `json:"status"` // "valid", "error", "skipped"
}

type AnalysisResult struct {
	Total   int              `json:"total"`
	Valid   int              `json:"valid"`
	Errors  int              `json:"errors"`
	Skipped int              `json:"skipped"`
	Results []PreloadResult  `json:"results"`
}
