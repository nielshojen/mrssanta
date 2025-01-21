package preflight

import "time"

// Device represents the data received in the request
type Device struct {
	SerialNumber         string    `firestore:"SerialNumber" json:"serial_num"`
	Hostname             string    `firestore:"Hostname" json:"hostname"`
	OSVersion            string    `firestore:"OSVersion" json:"os_version"`
	OSBuild              string    `firestore:"OSBuild" json:"os_build"`
	ModelIdentifier      string    `firestore:"ModelIdentifier,omitempty" json:"model_identifier,omitempty"`
	SantaVersion         string    `firestore:"SantaVersion" json:"santa_version"`
	PrimaryUser          string    `firestore:"PrimaryUser,omitempty" json:"primary_user,omitempty"`
	BinaryRuleCount      int       `firestore:"BinaryRuleCount,omitempty" json:"binary_rule_count,omitempty"`
	CertificateRuleCount int       `firestore:"CertificateRuleCount,omitempty" json:"certificate_rule_count,omitempty"`
	CompilerRuleCount    int       `firestore:"CompilerRuleCount,omitempty" json:"compiler_rule_count,omitempty"`
	TransitiveRuleCount  int       `firestore:"TransitiveRuleCount,omitempty" json:"transitive_rule_count,omitempty"`
	TeamIDRuleCount      int       `firestore:"TeamIDRuleCount,omitempty" json:"teamid_rule_count,omitempty"`
	SigningIDRuleCount   int       `firestore:"SigningIDRuleCount,omitempty" json:"signingid_rule_count,omitempty"`
	CDHashRuleCount      int       `firestore:"CDHashRuleCount,omitempty" json:"cdhash_rule_count,omitempty"`
	ClientMode           string    `firestore:"ClientMode" json:"client_mode"`
	RequestCleanSync     bool      `firestore:"RequestCleanSync,omitempty" json:"request_clean_sync,omitempty"`
	SyncCursor           string    `firestore:"SyncCursor,omitempty" json:"sync_cursor,omitempty"`
	SyncPage             int       `firestore:"SyncPage,omitempty" json:"sync_page,omitempty"`
	LastUpdated          time.Time `firestore:"LastUpdated,omitempty" json:"last_updated,omitempty"`
}

// PreflightResponse represents the response to a preflight request
type Response struct {
	EnableBundles            bool   `json:"enable_bundles,omitempty"`
	EnableTransitiveRules    bool   `json:"enable_transitive_rules,omitempty"`
	BatchSize                int    `json:"batch_size,omitempty"`
	FullSyncInterval         int    `json:"full_sync_interval,omitempty"`
	ClientMode               int    `json:"client_mode,omitempty"`
	AllowedPathRegEx         string `json:"allowed_path_regex,omitempty"`
	BlockedPathRegEx         string `json:"blocked_path_regex,omitempty"`
	BlockUSBMount            bool   `json:"block_usb_mount,omitempty"`
	RemountUSBMode           string `json:"remount_usb_mode,omitempty"`
	SyncType                 string `json:"sync_type,omitempty"`
	OverrideFileAccessAction string `json:"override_file_access_action,omitempty"`
}
