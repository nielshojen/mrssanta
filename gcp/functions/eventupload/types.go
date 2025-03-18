package eventupload

import "go.mongodb.org/mongo-driver/bson/primitive"

type Event struct {
	ID                          string             `bson:"_id,omitempty" json:"-"`
	FileSha256                  string             `bson:"file_sha256,omitempty" json:"file_sha256"`
	FilePath                    string             `bson:"file_path,omitempty" json:"file_path"`
	FileName                    string             `bson:"file_name,omitempty" json:"file_name"`
	ExecutingUser               string             `bson:"executing_user,omitempty" json:"executing_user,omitempty"`
	ExecutionTime               float64            `bson:"execution_time,omitempty" json:"execution_time,omitempty"`
	LoggedinUsers               []string           `bson:"loggedin_users,omitempty" json:"loggedin_users,omitempty"`
	CurrentSessions             []string           `bson:"current_sessions,omitempty" json:"current_sessions,omitempty"`
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
	PID                         int                `bson:"pid,omitempty" json:"pid,omitempty"`
	PPID                        int                `bson:"ppid,omitempty" json:"ppid,omitempty"`
	ParentName                  string             `bson:"parent_name,omitempty" json:"parent_name,omitempty"`
	QuarantineDataURL           string             `bson:"quarantine_data_url,omitempty" json:"quarantine_data_url,omitempty"`
	QuarantineRefererURL        string             `bson:"quarantine_referer_url,omitempty" json:"quarantine_referer_url,omitempty"`
	QuarantineTimestamp         float64            `bson:"quarantine_timestamp,omitempty" json:"quarantine_timestamp,omitempty"`
	QuarantineAgentBundleID     string             `bson:"quarantine_agent_bundle_id,omitempty" json:"quarantine_agent_bundle_id,omitempty"`
	SigningChain                []SigningChain     `bson:"signing_chain,omitempty" json:"signing_chain,omitempty"`
	SigningID                   string             `bson:"signing_id,omitempty" json:"signing_id,omitempty"`
	TeamID                      string             `bson:"team_id,omitempty" json:"team_id,omitempty"`
	CDHash                      string             `bson:"cdhash,omitempty" json:"cdhash,omitempty"`
	EntitlementInfo             []EntitlementInfo  `bson:"entitlementInfo,omitempty" json:"entitlementInfo,omitempty"`
	CSFlags                     int32              `bson:"csFlags,omitempty" json:"csFlags,omitempty"`
	SigningStatus               string             `bson:"signingStatus,omitempty" json:"signingStatus,omitempty"`
	VirusTotalResult            int                `bson:"virustotal_result,omitempty" json:"virustotal_result,omitempty"`
	CreationTime                primitive.DateTime `bson:"creation_time,omitempty" json:"creation_time,omitempty"`
	LastUpdated                 primitive.DateTime `bson:"last_updated,omitempty" json:"last_updated,omitempty"`
	Severity                    string             `json:"severity,omitempty"`                      // For GCP logging output
	Labels                      *Labels            `json:"logging.googleapis.com/labels,omitempty"` // For GCP logging output
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

type Labels struct {
	Env     string `json:"env,omitempty"`
	App     string `json:"app,omitempty"`
	Service string `json:"service,omitempty"`
	Owner   string `json:"owner,omitempty"`
	Team    string `json:"team,omitempty"`
	Version string `json:"version,omitempty"`
}

type Request struct {
	Events []Event `json:"events"`
}

type Response struct {
	EventUploadBundleBinaries []string `json:"event_upload_bundle_binaries"`
}
