package postflight

import (
	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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
	// PendingSyncType and SyncStartedAt are set by preflight for the sync that
	// this postflight completes.
	PendingSyncType string             `bson:"pending_sync_type,omitempty" json:"-"`
	SyncStartedAt   primitive.DateTime `bson:"sync_started_at,omitempty" json:"-"`
	// RequestedSyncType is a clean sync requested by an admin, cleared once a
	// clean sync completes.
	RequestedSyncType string `bson:"requested_sync_type,omitempty" json:"-"`
}

// Request is the part of a postflight request the server acts on.
type Request struct {
	RulesReceived  uint32
	RulesProcessed uint32
	MachineID      string
	// SyncType is the sync type the client reports it just completed.
	SyncType  string
	RulesHash string
}

func requestFromProto(req *syncv1.PostflightRequest) Request {
	syncType := ""
	if req.GetSyncType() != syncv1.SyncType_SYNC_TYPE_UNSPECIFIED {
		syncType = req.GetSyncType().String()
	}

	return Request{
		RulesReceived:  req.GetRulesReceived(),
		RulesProcessed: req.GetRulesProcessed(),
		MachineID:      req.GetMachineId(),
		SyncType:       syncType,
		RulesHash:      req.GetRulesHash(),
	}
}
