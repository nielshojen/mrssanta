package preflight

import (
	"strings"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	clientModeMonitor    uint32 = 1
	clientModeLockdown   uint32 = 2
	clientModeStandalone uint32 = 3
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
	// PendingSyncType and SyncStartedAt record the sync handed out by the most
	// recent preflight, so ruledownload and postflight act on the same sync.
	// They are server-owned and never read from the client.
	PendingSyncType string             `bson:"pending_sync_type,omitempty" json:"-"`
	SyncStartedAt   primitive.DateTime `bson:"sync_started_at,omitempty" json:"-"`
	// ReportedClientMode is the mode the agent said it was in at preflight,
	// which can differ from the assigned ClientMode until the sync completes.
	ReportedClientMode uint32 `bson:"reported_client_mode,omitempty" json:"-"`
	// RequestedSyncType is a clean sync requested by an admin. It is written
	// by the admin UI and cleared by postflight once a clean sync completes.
	RequestedSyncType string `bson:"requested_sync_type,omitempty" json:"-"`
	// Settings holds per-device overrides of the global preflight settings.
	// It is only written by the admin UI.
	Settings *Settings `bson:"settings,omitempty" json:"-"`
}

// deviceFromPreflight maps a preflight request onto the stored device fields.
// Server-owned fields (assigned client mode, sync state, settings) are left
// for the caller to fill in.
func deviceFromPreflight(req *syncv1.PreflightRequest, machineID string) Device {
	return Device{
		Identifier:            machineID,
		MachineID:             firstNonEmpty(req.GetMachineId(), machineID),
		SerialNumber:          req.GetSerialNumber(),
		Hostname:              req.GetHostname(),
		OSVersion:             req.GetOsVersion(),
		OSBuild:               req.GetOsBuild(),
		ModelIdentifier:       req.GetModelIdentifier(),
		SantaVersion:          req.GetSantaVersion(),
		PrimaryUser:           req.GetPrimaryUser(),
		PrimaryUserGroups:     req.GetPrimaryUserGroups(),
		PushNotificationToken: req.GetPushNotificationToken(),
		PushNotificationSync:  req.GetPushNotificationSync(),
		ReportedClientMode:    uint32(req.GetClientMode()),
		RequestCleanSync:      req.GetRequestCleanSync(),
		BinaryRuleCount:       req.GetBinaryRuleCount(),
		CertificateRuleCount:  req.GetCertificateRuleCount(),
		CompilerRuleCount:     req.GetCompilerRuleCount(),
		TransitiveRuleCount:   req.GetTransitiveRuleCount(),
		TeamIDRuleCount:       req.GetTeamidRuleCount(),
		SigningIDRuleCount:    req.GetSigningidRuleCount(),
		CDHashRuleCount:       req.GetCdhashRuleCount(),
		RulesHash:             req.GetRulesHash(),
		SIPStatus:             req.GetSipStatus(),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

// validClientMode reports whether mode is one the server can assign.
func validClientMode(mode uint32) bool {
	return mode == clientModeMonitor || mode == clientModeLockdown || mode == clientModeStandalone
}

// clientModeFromName converts a client mode name (as stored in settings) to
// its numeric value, returning ok=false for unknown names.
func clientModeFromName(name string) (uint32, bool) {
	value, ok := syncv1.ClientMode_value[strings.ToUpper(strings.TrimSpace(name))]
	if !ok || !validClientMode(uint32(value)) {
		return 0, false
	}
	return uint32(value), true
}
