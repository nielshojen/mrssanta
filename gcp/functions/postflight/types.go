package postflight

type Request struct {
	RulesReceived  int `json:"rules_received"`
	RulesProcessed int `json:"rules_processed"`
}

type Response struct {
	ID             string `json:"id"`
	RulesReceived  int    `json:"rules_received"`
	RulesProcessed int    `json:"rules_processed"`
}
