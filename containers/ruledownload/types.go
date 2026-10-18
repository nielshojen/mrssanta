package ruledownload

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Rule is a rule as stored in the "rules" collection. Only the fields that
// are part of the sync protocol are sent to Santa; see Rule.toProto.
type Rule struct {
	ID                     string             `bson:"_id,omitempty" json:"-"`
	Identifier             string             `bson:"identifier,omitempty" json:"identifier"`
	Policy                 string             `bson:"policy,omitempty" json:"policy"`
	RuleType               string             `bson:"rule_type,omitempty" json:"rule_type,omitempty"`
	CustomMessage          string             `bson:"custom_msg,omitempty" json:"custom_msg,omitempty"`
	CustomURL              string             `bson:"custom_url,omitempty" json:"custom_url,omitempty"`
	FileBundleBinaryCount  string             `bson:"file_bundle_binary_count,omitempty" json:"file_bundle_binary_count,omitempty"`
	FileBundleHash         string             `bson:"file_bundle_hash,omitempty" json:"file_bundle_hash,omitempty"`
	NotificationAppName    string             `bson:"notification_app_name,omitempty" json:"notification_app_name,omitempty"`
	CELExpr                string             `bson:"cel_expr,omitempty" json:"cel_expr,omitempty"`
	EventDetailButtonLabel string             `bson:"event_detail_button_label,omitempty" json:"event_detail_button_label,omitempty"`
	Scope                  string             `bson:"scope,omitempty" json:"scope"`
	Assigned               []string           `bson:"assigned,omitempty" json:"assigned,omitempty"`
	Converted              bool               `bson:"converted,omitempty" json:"converted,omitempty"`
	CreationTime           primitive.DateTime `bson:"creation_time,omitempty" json:"creation_time,omitempty"`
	LastUpdated            primitive.DateTime `bson:"last_updated,omitempty" json:"last_updated,omitempty"`
	// Deleted marks a tombstone: the rule was deleted and devices that still
	// have it receive a REMOVE on their next incremental sync.
	Deleted   bool               `bson:"deleted,omitempty" json:"-"`
	DeletedAt primitive.DateTime `bson:"deleted_at,omitempty" json:"-"`
	// RemovedFrom lists machines that were unassigned from a managedapp or
	// machine rule and still need a REMOVE for it.
	RemovedFrom []string `bson:"removed_from,omitempty" json:"-"`
}

type Device struct {
	ID                    string             `bson:"_id,omitempty" json:"-"`
	Identifier            string             `bson:"identifier" json:"identifier"`
	MachineID             string             `bson:"machine_id,omitempty" json:"machine_id,omitempty"`
	SerialNumber          string             `bson:"serial_num" json:"serial_num"`
	Hostname              string             `bson:"hostname" json:"hostname"`
	OSVersion             string             `bson:"os_version" json:"os_version"`
	OSBuild               string             `bson:"os_build" json:"os_build"`
	ModelIdentifier       string             `bson:"model_identifier,omitempty" json:"model_identifier,omitempty"`
	SantaVersion          string             `bson:"santa_version" json:"santa_version"`
	PrimaryUser           string             `bson:"primary_user,omitempty" json:"primary_user,omitempty"`
	PrimaryUserRole       string             `bson:"primary_user_role,omitempty" json:"primary_user_role,omitempty"`
	PrimaryUserGroups     []string           `bson:"primary_user_groups,omitempty" json:"primary_user_groups,omitempty"`
	PushNotificationToken string             `bson:"push_notification_token,omitempty" json:"push_notification_token,omitempty"`
	PushNotificationSync  bool               `bson:"push_notification_sync,omitempty" json:"push_notification_sync,omitempty"`
	ClientMode            uint32             `bson:"client_mode" json:"-"`
	RequestCleanSync      bool               `bson:"request_clean_sync,omitempty" json:"request_clean_sync,omitempty"`
	BinaryRuleCount       uint32             `bson:"binary_rule_count,omitempty" json:"binary_rule_count,omitempty"`
	CertificateRuleCount  uint32             `bson:"certificate_rule_count,omitempty" json:"certificate_rule_count,omitempty"`
	CompilerRuleCount     uint32             `bson:"compiler_rule_count,omitempty" json:"compiler_rule_count,omitempty"`
	TransitiveRuleCount   uint32             `bson:"transitive_rule_count,omitempty" json:"transitive_rule_count,omitempty"`
	TeamIDRuleCount       uint32             `bson:"teamid_rule_count,omitempty" json:"teamid_rule_count,omitempty"`
	SigningIDRuleCount    uint32             `bson:"signingid_rule_count,omitempty" json:"signingid_rule_count,omitempty"`
	CDHashRuleCount       uint32             `bson:"cdhash_rule_count,omitempty" json:"cdhash_rule_count,omitempty"`
	RuleCount             uint32             `bson:"rule_count,omitempty" json:"-"`
	RulesHash             string             `bson:"rules_hash,omitempty" json:"rulesHash,omitempty"`
	SIPStatus             uint32             `bson:"sip_status,omitempty" json:"sip_status,omitempty"`
	NeedsCleanSync        bool               `bson:"needs_clean_sync,omitempty" json:"needs_clean_sync,omitempty"`
	SyncInterval          uint32             `bson:"sync_interval,omitempty" json:"sync_interval,omitempty"`
	LastSync              primitive.DateTime `bson:"last_sync,omitempty" json:"last_sync,omitempty"`
	LastCleanSync         primitive.DateTime `bson:"last_clean_sync,omitempty" json:"last_clean_sync,omitempty"`
	LastUpdated           primitive.DateTime `bson:"last_updated,omitempty" json:"last_updated,omitempty"`
	// PendingSyncType is the sync type handed out by the most recent preflight.
	PendingSyncType string `bson:"pending_sync_type,omitempty" json:"-"`
}

type CursorMetadata struct {
	Identifier  string `json:"identifier"`
	LastUpdated int64  `json:"last_updated"`
}
