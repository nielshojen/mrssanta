package ruledownload

import "time"

type Rule struct {
	Identifier            string `firestore:"identifier" json:"identifier"`
	Policy                string `firestore:"policy" json:"policy"`
	RuleType              string `firestore:"rule_type" json:"rule_type"`
	CustomMessage         string `firestore:"custom_msg" json:"custom_msg,omitempty"`
	CustomURL             string `firestore:"custom_url" json:"custom_url,omitempty"`
	CreationTime          string `firestore:"creation_time" json:"-"`
	FileBundleBinaryCount string `firestore:"file_bundle_binary_count" json:"file_bundle_binary_count,omitempty"`
	FileBundleHash        string `firestore:"file_bundle_hash" json:"file_bundle_hash,omitempty"`
	Scope                 string `firestore:"scope" json:"-"`
	Assigned              string `firestore:"assigned" json:"-"`
}

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

type Response struct {
	Cursor string  `json:"cursor,omitempty"`
	Rules  []*Rule `json:"rules,omitempty"`
}
