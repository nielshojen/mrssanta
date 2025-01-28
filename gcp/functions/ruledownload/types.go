package ruledownload

import "time"

type Rule struct {
	Identifier            string   `firestore:"Identifier" json:"identifier"`
	Policy                string   `firestore:"Policy" json:"policy"`
	RuleType              string   `firestore:"RuleType" json:"rule_type"`
	CustomMessage         string   `firestore:"CustomMessage" json:"custom_msg,omitempty"`
	CustomURL             string   `firestore:"CustomURL" json:"custom_url,omitempty"`
	CreationTime          string   `firestore:"CreationTime" json:"-"`
	FileBundleBinaryCount string   `firestore:"FileBundleBinaryCount" json:"file_bundle_binary_count,omitempty"`
	FileBundleHash        string   `firestore:"FileBundleHash" json:"file_bundle_hash,omitempty"`
	Scope                 string   `firestore:"Scope" json:"-"`
	Assigned              []string `firestore:"Assigned" json:"-"`
}

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

type Response struct {
	Cursor string  `json:"cursor,omitempty"`
	Rules  []*Rule `json:"rules,omitempty"`
}
