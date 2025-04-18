package ruledownload

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Request struct {
	Cursor string `json:"cursor,omitempty"`
}

type Rule struct {
	ID                    string             `bson:"_id,omitempty" json:"-"`
	Identifier            string             `bson:"identifier,omitempty" json:"identifier"`
	Policy                string             `bson:"policy,omitempty" json:"policy"`
	RuleType              string             `bson:"rule_type,omitempty" json:"rule_type,omitempty"`
	CustomMessage         string             `bson:"custom_msg,omitempty" json:"custom_msg,omitempty"`
	CustomURL             string             `bson:"custom_url,omitempty" json:"custom_url,omitempty"`
	FileBundleBinaryCount string             `bson:"file_bundle_binary_count,omitempty" json:"file_bundle_binary_count,omitempty"`
	FileBundleHash        string             `bson:"file_bundle_hash,omitempty" json:"file_bundle_hash,omitempty"`
	Scope                 string             `bson:"scope,omitempty" json:"scope"`
	Assigned              []string           `bson:"assigned,omitempty" json:"assigned,omitempty"`
	CreationTime          primitive.DateTime `bson:"creation_time,omitempty" json:"creation_time,omitempty"`
	LastUpdated           primitive.DateTime `bson:"last_updated,omitempty" json:"last_updated,omitempty"`
}

type Device struct {
	ID                   string             `bson:"_id,omitempty" json:"-"`
	Identifier           string             `bson:"identifier" json:"-"`
	SerialNumber         string             `bson:"serial_num" json:"serial_num"`
	Hostname             string             `bson:"hostname" json:"hostname"`
	OSVersion            string             `bson:"os_version" json:"os_version"`
	OSBuild              string             `bson:"os_build" json:"os_build"`
	ModelIdentifier      string             `bson:"model_identifier,omitempty" json:"model_identifier,omitempty"`
	SantaVersion         string             `bson:"santa_version" json:"santa_version"`
	PrimaryUser          string             `bson:"primary_user" json:"primary_user"`
	BinaryRuleCount      int                `bson:"binary_rule_count,omitempty" json:"binary_rule_count,omitempty"`
	CertificateRuleCount int                `bson:"certificate_rule_count,omitempty" json:"certificate_rule_count,omitempty"`
	CompilerRuleCount    int                `bson:"compiler_rule_count,omitempty" json:"compiler_rule_count,omitempty"`
	TransitiveRuleCount  int                `bson:"transitive_rule_count,omitempty" json:"transitive_rule_count,omitempty"`
	TeamIDRuleCount      int                `bson:"teamid_rule_count,omitempty" json:"teamid_rule_count,omitempty"`
	SigningIDRuleCount   int                `bson:"signingid_rule_count,omitempty" json:"signingid_rule_count,omitempty"`
	CDHashRuleCount      int                `bson:"cdhash_rule_count,omitempty" json:"cdhash_rule_count,omitempty"`
	ClientMode           int                `bson:"client_mode" json:"-"`
	RequestCleanSync     bool               `bson:"request_clean_sync,omitempty" json:"request_clean_sync,omitempty"`
	NeedsCleanSync       bool               `bson:"needs_clean_sync,omitempty" json:"needs_clean_sync,omitempty"`
	LastCleanSync        primitive.DateTime `bson:"last_clean_sync,omitempty" json:"last_clean_sync,omitempty"`
	LastUpdated          primitive.DateTime `bson:"last_updated,omitempty" json:"last_updated,omitempty"`
}

type CursorMetadata struct {
	Identifier string `json:"identifier"`
}

type Response struct {
	Cursor string  `json:"cursor,omitempty"`
	Rules  []*Rule `json:"rules,omitempty"`
}
