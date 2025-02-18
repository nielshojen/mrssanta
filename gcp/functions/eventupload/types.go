package eventupload

import "time"

type Event struct {
	FileSha256                  string            `firestore:"FileSha256" json:"file_sha256"`
	FilePath                    string            `firestore:"FilePath" json:"file_path"`
	FileName                    string            `firestore:"FileName" json:"file_name"`
	ExecutingUser               string            `firestore:"ExecutingUser,omitempty" json:"executing_user,omitempty"`
	ExecutionTime               float64           `firestore:"ExecutionTime,omitempty" json:"execution_time,omitempty"`
	LoggedinUsers               []string          `firestore:"LoggedinUsers,omitempty" json:"loggedin_users,omitempty"`
	CurrentSessions             []string          `firestore:"CurrentSessions,omitempty" json:"current_sessions,omitempty"`
	Decision                    string            `firestore:"Decision" json:"decision"`
	FileBundleID                string            `firestore:"FileBundleID,omitempty" json:"file_bundle_id,omitempty"`
	FileBundlePath              string            `firestore:"FileBundlePath,omitempty" json:"file_bundle_path,omitempty"`
	FileBundleExecutableRelPath string            `firestore:"FileBundleExecutableRelPath,omitempty" json:"file_bundle_executable_rel_path,omitempty"`
	FileBundleName              string            `firestore:"FileBundleName,omitempty" json:"file_bundle_name,omitempty"`
	FileBundleVersion           string            `firestore:"FileBundleVersion,omitempty" json:"file_bundle_version,omitempty"`
	FileBundleVersionString     string            `firestore:"FileBundleVersionString,omitempty" json:"file_bundle_version_string,omitempty"`
	FileBundleHash              string            `firestore:"FileBundleHash,omitempty" json:"file_bundle_hash,omitempty"`
	FileBundleHashMillis        float64           `firestore:"FileBundleHashMillis,omitempty" json:"file_bundle_hash_millis,omitempty"`
	FileBundleBinaryCount       int               `firestore:"FileBundleBinaryCount,omitempty" json:"file_bundle_binary_count,omitempty"`
	PID                         int               `firestore:"PID,omitempty" json:"pid,omitempty"`
	PPID                        int               `firestore:"PPID,omitempty" json:"ppid,omitempty"`
	ParentName                  string            `firestore:"ParentName,omitempty" json:"parent_name,omitempty"`
	QuarantineDataURL           string            `firestore:"QuarantineDataURL,omitempty" json:"quarantine_data_url,omitempty"`
	QuarantineRefererURL        string            `firestore:"QuarantineRefererURL,omitempty" json:"quarantine_referer_url,omitempty"`
	QuarantineTimestamp         float64           `firestore:"QuarantineTimestamp,omitempty" json:"quarantine_timestamp,omitempty"`
	QuarantineAgentBundleID     string            `firestore:"QuarantineAgentBundleID,omitempty" json:"quarantine_agent_bundle_id,omitempty"`
	SigningChain                []SigningChain    `firestore:"SigningChain,omitempty" json:"signing_chain,omitempty"`
	SigningID                   string            `firestore:"SigningID,omitempty" json:"signing_id,omitempty"`
	TeamID                      string            `firestore:"TeamID,omitempty" json:"team_id,omitempty"`
	CDHash                      string            `firestore:"CDHash,omitempty" json:"cdhash,omitempty"`
	EntitlementInfo             []EntitlementInfo `firestore:"EntitlementInfo,omitempty" json:"entitlementInfo,omitempty"`
	CSFlags                     int32             `firestore:"CSFlags,omitempty" json:"csflags,omitempty"`
	SigningStatus               string            `firestore:"SigningStatus,omitempty" json:"signingStatus,omitempty"`
	VirusTotalResult            int               `firestore:"VirusTotalResult,omitempty" json:"virustotalresult,omitempty"`
	LastUpdated                 time.Time         `firestore:"LastUpdated,omitempty"`
	Severity                    string            `json:"severity,omitempty"`                      // For GCP logging output
	Labels                      *Labels           `json:"logging.googleapis.com/labels,omitempty"` // For GCP logging output
}

type SigningChain struct {
	Sha256     string `json:"sha256"`
	CN         string `json:"cn"`
	Org        string `json:"org"`
	OU         string `json:"ou"`
	ValidFrom  int    `json:"valid_from"`
	ValidUntil int    `json:"valid_until"`
}

type EntitlementInfo struct {
	EntitlementsFiltered string        `json:"entitlementsFiltered"`
	Entitlements         []Entitlement `json:"entitlements"`
}

type Entitlement struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
