package datatypes

// example:

type VictoriaMetricsImport struct {
	// required key __name__. Rest are treated as labels
	Metric     map[string]string `json:"metric"`
	Values     []float64         `json:"values"`
	Timestamps []int64           `json:"timestamps"` // milliseconds
}
