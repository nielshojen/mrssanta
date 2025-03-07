package ruledownload

import "go.mongodb.org/mongo-driver/bson/primitive"

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

type Response struct {
	Cursor string  `json:"cursor,omitempty"`
	Rules  []*Rule `json:"rules,omitempty"`
}
