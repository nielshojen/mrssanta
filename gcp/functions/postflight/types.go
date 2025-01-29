package postflight

type Request struct {
	RulesReceived  int `json:"rules_received"`
	RulesProcessed int `json:"rules_processed"`
}
