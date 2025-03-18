package postflight

type Request struct {
	RulesReceived  int    `json:"rules_received"`
	RulesProcessed int    `json:"rules_processed"`
	MachineID      string `json:"machine_id"`
	SyncType       int    `json:"sync_type"`
}
