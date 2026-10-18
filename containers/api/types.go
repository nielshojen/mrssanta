package api

import "go.mongodb.org/mongo-driver/v2/bson"

type Rule struct {
	ID                     string        `bson:"_id,omitempty" json:"-"`
	Identifier             string        `bson:"identifier,omitempty" json:"identifier"`
	Policy                 string        `bson:"policy,omitempty" json:"policy"`
	RuleType               string        `bson:"rule_type,omitempty" json:"rule_type"`
	CustomMessage          string        `bson:"custom_msg,omitempty" json:"custom_msg,omitempty"`
	CustomURL              string        `bson:"custom_url,omitempty" json:"custom_url,omitempty"`
	FileBundleBinaryCount  string        `bson:"file_bundle_binary_count,omitempty" json:"file_bundle_binary_count,omitempty"`
	FileBundleHash         string        `bson:"file_bundle_hash,omitempty" json:"file_bundle_hash,omitempty"`
	NotificationAppName    string        `bson:"notification_app_name,omitempty" json:"notification_app_name,omitempty"`
	CELExpr                string        `bson:"cel_expr,omitempty" json:"cel_expr,omitempty"`
	EventDetailButtonLabel string        `bson:"event_detail_button_label,omitempty" json:"event_detail_button_label,omitempty"`
	Scope                  string        `bson:"scope,omitempty" json:"scope"`
	Assigned               []string      `bson:"assigned,omitempty" json:"assigned,omitempty"`
	Converted              bool          `bson:"converted,omitempty" json:"converted,omitempty"`
	CreationTime           bson.DateTime `bson:"creation_time,omitempty" json:"creation_time,omitempty"`
	LastUpdated            bson.DateTime `bson:"last_updated,omitempty" json:"last_updated,omitempty"`
}

type Device struct {
	ID                    string        `bson:"_id,omitempty" json:"-"`
	Identifier            string        `bson:"identifier" json:"identifier"`
	MachineID             string        `bson:"machine_id,omitempty" json:"machine_id,omitempty"`
	SerialNumber          string        `bson:"serial_num" json:"serial_num"`
	Hostname              string        `bson:"hostname" json:"hostname"`
	OSVersion             string        `bson:"os_version" json:"os_version"`
	OSBuild               string        `bson:"os_build" json:"os_build"`
	ModelIdentifier       string        `bson:"model_identifier,omitempty" json:"model_identifier,omitempty"`
	SantaVersion          string        `bson:"santa_version" json:"santa_version"`
	PrimaryUser           string        `bson:"primary_user,omitempty" json:"primary_user,omitempty"`
	PrimaryUserRole       string        `bson:"primary_user_role,omitempty" json:"primary_user_role,omitempty"`
	PrimaryUserGroups     []string      `bson:"primary_user_groups,omitempty" json:"primary_user_groups,omitempty"`
	PushNotificationToken string        `bson:"push_notification_token,omitempty" json:"push_notification_token,omitempty"`
	PushNotificationSync  bool          `bson:"push_notification_sync,omitempty" json:"push_notification_sync,omitempty"`
	ClientMode            uint32        `bson:"client_mode" json:"client_mode"`
	RequestCleanSync      bool          `bson:"request_clean_sync,omitempty" json:"request_clean_sync,omitempty"`
	BinaryRuleCount       uint32        `bson:"binary_rule_count,omitempty" json:"binary_rule_count,omitempty"`
	CertificateRuleCount  uint32        `bson:"certificate_rule_count,omitempty" json:"certificate_rule_count,omitempty"`
	CompilerRuleCount     uint32        `bson:"compiler_rule_count,omitempty" json:"compiler_rule_count,omitempty"`
	TransitiveRuleCount   uint32        `bson:"transitive_rule_count,omitempty" json:"transitive_rule_count,omitempty"`
	TeamIDRuleCount       uint32        `bson:"teamid_rule_count,omitempty" json:"teamid_rule_count,omitempty"`
	SigningIDRuleCount    uint32        `bson:"signingid_rule_count,omitempty" json:"signingid_rule_count,omitempty"`
	CDHashRuleCount       uint32        `bson:"cdhash_rule_count,omitempty" json:"cdhash_rule_count,omitempty"`
	RuleCount             uint32        `bson:"rule_count,omitempty" json:"-"`
	RulesHash             string        `bson:"rules_hash,omitempty" json:"rulesHash,omitempty"`
	SIPStatus             uint32        `bson:"sip_status,omitempty" json:"sip_status,omitempty"`
	NeedsCleanSync        bool          `bson:"needs_clean_sync,omitempty" json:"needs_clean_sync,omitempty"`
	SyncInterval          uint32        `bson:"sync_interval,omitempty" json:"sync_interval,omitempty"`
	LastSync              bson.DateTime `bson:"last_sync,omitempty" json:"last_sync,omitempty"`
	LastCleanSync         bson.DateTime `bson:"last_clean_sync,omitempty" json:"last_clean_sync,omitempty"`
	LastUpdated           bson.DateTime `bson:"last_updated,omitempty" json:"last_updated,omitempty"`
}

type Event struct {
	ID                          string           `bson:"_id,omitempty" json:"-"`
	FileSha256                  string           `bson:"file_sha256,omitempty" json:"file_sha256"`
	FilePath                    string           `bson:"file_path,omitempty" json:"file_path"`
	FileName                    string           `bson:"file_name,omitempty" json:"file_name"`
	ExecutingUser               string           `bson:"executing_user,omitempty" json:"executing_user,omitempty"`
	ExecutionTime               float64          `bson:"execution_time,omitempty" json:"execution_time,omitempty"`
	LoggedinUsers               []string         `bson:"loggedin_users,omitempty" json:"loggedin_users,omitempty"`
	CurrentSessions             []string         `bson:"current_sessions,omitempty" json:"current_sessions,omitempty"`
	Decision                    string           `bson:"decision,omitempty" json:"decision"`
	FileBundleID                string           `bson:"file_bundle_id,omitempty" json:"file_bundle_id,omitempty"`
	FileBundlePath              string           `bson:"file_bundle_path,omitempty" json:"file_bundle_path,omitempty"`
	FileBundleExecutableRelPath string           `bson:"file_bundle_executable_rel_path,omitempty" json:"file_bundle_executable_rel_path,omitempty"`
	FileBundleName              string           `bson:"file_bundle_name,omitempty" json:"file_bundle_name,omitempty"`
	FileBundleVersion           string           `bson:"file_bundle_version,omitempty" json:"file_bundle_version,omitempty"`
	FileBundleVersionString     string           `bson:"file_bundle_version_string,omitempty" json:"file_bundle_version_string,omitempty"`
	FileBundleHash              string           `bson:"file_bundle_hash,omitempty" json:"file_bundle_hash,omitempty"`
	FileBundleHashMillis        float64          `bson:"file_bundle_hash_millis,omitempty" json:"file_bundle_hash_millis,omitempty"`
	FileBundleBinaryCount       int              `bson:"file_bundle_binary_count,omitempty" json:"file_bundle_binary_count,omitempty"`
	PID                         int              `bson:"pid,omitempty" json:"pid,omitempty"`
	PPID                        int              `bson:"ppid,omitempty" json:"ppid,omitempty"`
	ParentName                  string           `bson:"parent_name,omitempty" json:"parent_name,omitempty"`
	QuarantineDataURL           string           `bson:"quarantine_data_url,omitempty" json:"quarantine_data_url,omitempty"`
	QuarantineRefererURL        string           `bson:"quarantine_referer_url,omitempty" json:"quarantine_referer_url,omitempty"`
	QuarantineTimestamp         float64          `bson:"quarantine_timestamp,omitempty" json:"quarantine_timestamp,omitempty"`
	QuarantineAgentBundleID     string           `bson:"quarantine_agent_bundle_id,omitempty" json:"quarantine_agent_bundle_id,omitempty"`
	SigningChain                []SigningChain   `bson:"signing_chain,omitempty" json:"signing_chain,omitempty"`
	SigningID                   string           `bson:"signing_id,omitempty" json:"signing_id,omitempty"`
	TeamID                      string           `bson:"team_id,omitempty" json:"team_id,omitempty"`
	CDHash                      string           `bson:"cdhash,omitempty" json:"cdhash,omitempty"`
	EntitlementInfo             *EntitlementInfo `bson:"entitlementInfo,omitempty" json:"entitlementInfo,omitempty"`
	CSFlags                     int32            `bson:"csFlags,omitempty" json:"csFlags,omitempty"`
	SigningStatus               string           `bson:"SigningStatus,omitempty" json:"signingStatus,omitempty"`
	VirusTotalResult            int              `bson:"virustotal_result,omitempty" json:"virustotal_result,omitempty"`
	CreationTime                bson.DateTime    `bson:"creation_time,omitempty" json:"creation_time,omitempty"`
	LastUpdated                 bson.DateTime    `bson:"last_updated,omitempty" json:"last_updated,omitempty"`
}

type SigningChain struct {
	Sha256     string `bson:"sha256" json:"sha256"`
	CN         string `bson:"cn" json:"cn"`
	Org        string `bson:"org" json:"org"`
	OU         string `bson:"ou" json:"ou"`
	ValidFrom  int    `bson:"valid_from" json:"valid_from"`
	ValidUntil int    `bson:"valid_until" json:"valid_until"`
}

type EntitlementInfo struct {
	EntitlementsFiltered string        `bson:"entitlementsFiltered" json:"entitlementsFiltered"`
	Entitlements         []Entitlement `bson:"entitlements" json:"entitlements"`
}

type Entitlement struct {
	Key   string `bson:"key" json:"key"`
	Value string `bson:"value" json:"value"`
}
