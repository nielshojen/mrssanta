package eventupload

import "go.mongodb.org/mongo-driver/bson/primitive"

// Event is an execution event as stored in the "events" collection (one
// document per binary) and as shipped to the logging pipeline.
type Event struct {
	ID                          string             `bson:"_id,omitempty" json:"-"`
	FileSha256                  string             `bson:"file_sha256,omitempty" json:"file_sha256"`
	FilePath                    string             `bson:"file_path,omitempty" json:"file_path"`
	FileName                    string             `bson:"file_name,omitempty" json:"file_name"`
	ExecutingUser               string             `bson:"executing_user,omitempty" json:"executing_user,omitempty"`
	ExecutionTime               float64            `bson:"execution_time,omitempty" json:"execution_time,omitempty"`
	LoggedinUsers               []string           `bson:"logged_in_users,omitempty" json:"logged_in_users,omitempty"`
	CurrentSessions             []string           `bson:"current_sessions,omitempty" json:"current_sessions,omitempty"`
	PID                         int                `bson:"pid,omitempty" json:"pid,omitempty"`
	PPID                        int                `bson:"ppid,omitempty" json:"ppid,omitempty"`
	Decision                    string             `bson:"decision,omitempty" json:"decision"`
	FileBundleID                string             `bson:"file_bundle_id,omitempty" json:"file_bundle_id,omitempty"`
	FileBundlePath              string             `bson:"file_bundle_path,omitempty" json:"file_bundle_path,omitempty"`
	FileBundleExecutableRelPath string             `bson:"file_bundle_executable_rel_path,omitempty" json:"file_bundle_executable_rel_path,omitempty"`
	FileBundleName              string             `bson:"file_bundle_name,omitempty" json:"file_bundle_name,omitempty"`
	FileBundleVersion           string             `bson:"file_bundle_version,omitempty" json:"file_bundle_version,omitempty"`
	FileBundleVersionString     string             `bson:"file_bundle_version_string,omitempty" json:"file_bundle_version_string,omitempty"`
	FileBundleHash              string             `bson:"file_bundle_hash,omitempty" json:"file_bundle_hash,omitempty"`
	FileBundleHashMillis        float64            `bson:"file_bundle_hash_millis,omitempty" json:"file_bundle_hash_millis,omitempty"`
	FileBundleBinaryCount       int                `bson:"file_bundle_binary_count,omitempty" json:"file_bundle_binary_count,omitempty"`
	ParentName                  string             `bson:"parent_name,omitempty" json:"parent_name,omitempty"`
	QuarantineDataURL           string             `bson:"quarantine_data_url,omitempty" json:"quarantine_data_url,omitempty"`
	QuarantineRefererURL        string             `bson:"quarantine_referer_url,omitempty" json:"quarantine_referer_url,omitempty"`
	QuarantineTimestamp         float64            `bson:"quarantine_timestamp,omitempty" json:"quarantine_timestamp,omitempty"`
	QuarantineAgentBundleID     string             `bson:"quarantine_agent_bundle_id,omitempty" json:"quarantine_agent_bundle_id,omitempty"`
	SigningChain                []SigningChain     `bson:"signing_chain,omitempty" json:"signing_chain,omitempty"`
	SigningID                   string             `bson:"signing_id,omitempty" json:"signing_id,omitempty"`
	TeamID                      string             `bson:"team_id,omitempty" json:"team_id,omitempty"`
	CDHash                      string             `bson:"cdhash,omitempty" json:"cdhash,omitempty"`
	EntitlementInfo             *EntitlementInfo   `bson:"entitlementInfo,omitempty" json:"entitlementInfo,omitempty"`
	CSFlags                     int64              `bson:"csFlags,omitempty" json:"csFlags,omitempty"`
	SigningStatus               string             `bson:"signingStatus,omitempty" json:"signingStatus,omitempty"`
	SecureSigningTime           uint32             `bson:"secure_signing_time,omitempty" json:"secure_signing_time,omitempty"`
	SigningTime                 uint32             `bson:"signing_time,omitempty" json:"signing_time,omitempty"`
	StaticRule                  bool               `bson:"static_rule,omitempty" json:"static_rule,omitempty"`
	IdentityUnverified          bool               `bson:"identity_unverified,omitempty" json:"identity_unverified,omitempty"`
	VirusTotalResult            int                `bson:"virustotal_result,omitempty" json:"virustotal_result,omitempty"`
	CreationTime                primitive.DateTime `bson:"creation_time,omitempty" json:"creation_time,omitempty"`
	LastUpdated                 primitive.DateTime `bson:"last_updated,omitempty" json:"last_updated,omitempty"`
	Severity                    string             `json:"severity,omitempty"`
	Labels                      *Labels            `json:"logging.googleapis.com/labels,omitempty"`
	Service                     string             `json:"service,omitempty"`
	DDSource                    string             `json:"ddsource,omitempty"`
	DDTags                      string             `json:"ddtags,omitempty"`
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
	EntitlementsFiltered bool          `bson:"entitlementsFiltered,omitempty" json:"entitlementsFiltered,omitempty"`
	Entitlements         []Entitlement `bson:"entitlements,omitempty" json:"entitlements,omitempty"`
}

type Entitlement struct {
	Key   string `bson:"key,omitempty" json:"key,omitempty"`
	Value string `bson:"value,omitempty" json:"value,omitempty"`
}

type Labels struct {
	Env     string `json:"env,omitempty"`
	App     string `json:"app,omitempty"`
	Service string `json:"service,omitempty"`
	Owner   string `json:"owner,omitempty"`
	Team    string `json:"team,omitempty"`
	Version string `json:"version,omitempty"`
}

// FileAccessEvent is a File Access Authorization violation as stored in the
// "file_access_events" collection (one document per occurrence).
type FileAccessEvent struct {
	MachineID   string  `bson:"machine_id"`
	RuleName    string  `bson:"rule_name"`
	RuleVersion string  `bson:"rule_version,omitempty"`
	Target      string  `bson:"target"`
	Decision    string  `bson:"decision"`
	AccessTime  float64 `bson:"access_time,omitempty"`
	// ProcessChain starts with the process that made the access, followed by
	// its ancestors.
	ProcessChain []Process          `bson:"process_chain,omitempty"`
	CreationTime primitive.DateTime `bson:"creation_time"`
}

type Process struct {
	FilePath     string         `bson:"file_path,omitempty"`
	CDHash       string         `bson:"cdhash,omitempty"`
	FileSHA256   string         `bson:"file_sha256,omitempty"`
	SigningID    string         `bson:"signing_id,omitempty"`
	TeamID       string         `bson:"team_id,omitempty"`
	PID          int32          `bson:"pid,omitempty"`
	SigningChain []SigningChain `bson:"signing_chain,omitempty"`
}

// StandaloneApproval records a rule a user created locally in STANDALONE
// mode, stored in the "standalone_approvals" collection.
type StandaloneApproval struct {
	MachineID    string             `bson:"machine_id"`
	Identifier   string             `bson:"identifier"`
	Decision     string             `bson:"decision"`
	Timestamp    primitive.DateTime `bson:"timestamp,omitempty"`
	CreationTime primitive.DateTime `bson:"creation_time"`
}
