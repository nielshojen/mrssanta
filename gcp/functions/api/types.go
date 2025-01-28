package api

type Rule struct {
	Identifier            string   `firestore:"Identifier" json:"identifier"`
	Policy                string   `firestore:"Policy" json:"policy"`
	RuleType              string   `firestore:"RuleType" json:"rule_type"`
	CustomMessage         string   `firestore:"CustomMessage" json:"custom_msg,omitempty"`
	CustomURL             string   `firestore:"CustomURL" json:"custom_url,omitempty"`
	CreationTime          string   `firestore:"CreationTime" json:"creation_time,omitempty"`
	FileBundleBinaryCount string   `firestore:"FileBundleBinaryCount" json:"file_bundle_binary_count,omitempty"`
	FileBundleHash        string   `firestore:"FileBundleHash" json:"file_bundle_hash,omitempty"`
	Scope                 string   `firestore:"Scope" json:"scope"`
	Assigned              []string `firestore:"Assigned" json:"assigned,omitempty"`
}
