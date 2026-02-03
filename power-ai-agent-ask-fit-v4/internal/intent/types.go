package intent

type Result struct {
	Intent     string
	AgentCode  string
	Router     string
	Confidence float64
	Reason     string
	IsAsk      bool
}
