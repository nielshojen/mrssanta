package ruledownload

import "time"

type Rule struct {
	Identifier            string    `firestore:"Identifier" json:"identifier"`
	Policy                string    `firestore:"Policy" json:"policy"`
	RuleType              string    `firestore:"RuleType" json:"rule_type"`
	CustomMessage         string    `firestore:"CustomMessage" json:"custom_msg,omitempty"`
	CustomURL             string    `firestore:"CustomURL" json:"custom_url,omitempty"`
	CreationTime          time.Time `firestore:"CreationTime,serverTimestamp,omitempty" json:"-"`
	FileBundleBinaryCount string    `firestore:"FileBundleBinaryCount" json:"file_bundle_binary_count,omitempty"`
	FileBundleHash        string    `firestore:"FileBundleHash" json:"file_bundle_hash,omitempty"`
	Scope                 string    `firestore:"Scope" json:"-"`
	Assigned              []string  `firestore:"Assigned" json:"-"`
	LastUpdated           time.Time `firestore:"LastUpdated,omitempty" json:"-"`
}

type Response struct {
	Cursor string  `json:"cursor,omitempty"`
	Rules  []*Rule `json:"rules,omitempty"`
}
