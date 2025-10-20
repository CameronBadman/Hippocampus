package handlers

type InsertRequest struct {
	AgentID string `json:"agent_id"`
	Key     string `json:"key"`
	Text    string `json:"text"`
}

type SearchRequest struct {
	AgentID   string  `json:"agent_id"`
	Text      string  `json:"text"`
	Epsilon   float32 `json:"epsilon"`
	Threshold float32 `json:"threshold"`
	TopK      int     `json:"top_k"`
}

type InsertCSVRequest struct {
	AgentID string `json:"agent_id"`
	CSVFile string `json:"csv_file"`
}

type Response struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
