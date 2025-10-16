package types

import "encoding/json"

type Metric struct {
	Pod        string  `json:"pod"`
	CPUPercent float64 `json:"cpu_percent"`
	MemMB      float64 `json:"mem_mb"`
	Timestamp  int64   `json:"timestamp"`
	Tenant     string  `json:"tenant"`
}

type Action struct {
	Pod       string `json:"pod"`
	Action    string `json:"action"`
	Timestamp int64  `json:"timestamp"`
	Tenant    string `json:"tenant"`
}

type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
