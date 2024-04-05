package preflight

import "time"

// Device represents the data received in the request
type Device struct {
	SerialNumber         string    `firestore:"serial_num" json:"serial_num"`
	Hostname             string    `firestore:"hostname" json:"hostname"`
	OSVersion            string    `firestore:"os_version" json:"os_version"`
	OSBuild              string    `firestore:"os_build" json:"os_build"`
	ModelIdentifier      string    `firestore:"model_identifier,omitempty" json:"model_identifier,omitempty"`
	SantaVersion         string    `firestore:"santa_version" json:"santa_version"`
	PrimaryUser          string    `firestore:"primary_user" json:"primary_user"`
	BinaryRuleCount      int       `firestore:"binary_rule_count,omitempty" json:"binary_rule_count,omitempty"`
	CertificateRuleCount int       `firestore:"certificate_rule_count,omitempty" json:"certificate_rule_count,omitempty"`
	CompilerRuleCount    int       `firestore:"compiler_rule_count,omitempty" json:"compiler_rule_count,omitempty"`
	TransitiveRuleCount  int       `firestore:"transitive_rule_count,omitempty" json:"transitive_rule_count,omitempty"`
	TeamIDRuleCount      int       `firestore:"teamid_rule_count,omitempty" json:"teamid_rule_count,omitempty"`
	SigningIDRuleCount   int       `firestore:"signingid_rule_count,omitempty" json:"signingid_rule_count,omitempty"`
	CDHashRuleCount      int       `firestore:"cdhash_rule_count,omitempty" json:"cdhash_rule_count,omitempty"`
	ClientMode           string    `firestore:"client_mode" json:"client_mode"`
	RequestCleanSync     bool      `firestore:"request_clean_sync,omitempty" json:"request_clean_sync,omitempty"`
	SyncCursor           string    `firestore:"sync_cursor,omitempty" json:"sync_cursor,omitempty"`
	SyncPage             int       `firestore:"sync_page,omitempty" json:"sync_page,omitempty"`
	LastUpdated          time.Time `firestore:"last_updated,omitempty" json:"last_updated,omitempty"`
}

// PreflightResponse represents the response to a preflight request
type Response struct {
	EnableBundles            bool   `json:"enable_bundles,omitempty"`
	EnableTransitiveRules    bool   `json:"enable_transitive_rules,omitempty"`
	BatchSize                int    `json:"batch_size,omitempty"`
	FullSyncInterval         int    `json:"full_sync_interval,omitempty"`
	ClientMode               string `json:"client_mode,omitempty"`
	AllowedPathRegEx         string `json:"allowed_path_regex,omitempty"`
	BlockedPathRegEx         string `json:"blocked_path_regex,omitempty"`
	BlockUSBMount            bool   `json:"block_usb_mount,omitempty"`
	RemountUSBMode           string `json:"remount_usb_mode,omitempty"`
	SyncType                 string `json:"sync_type,omitempty"`
	OverrideFileAccessAction string `json:"override_file_access_action,omitempty"`
}
