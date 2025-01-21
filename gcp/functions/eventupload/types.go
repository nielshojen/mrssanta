package eventupload

import "time"

type Event struct {
	FileSha256                  string         `firestore:"FileSha256" json:"file_sha256"`
	FilePath                    string         `firestore:"FilePath" json:"file_path"`
	FileName                    string         `firestore:"FileName" json:"file_name"`
	ExecutingUser               string         `firestore:"ExecutingUser" json:"executing_user,omitempty"`
	ExecutionTime               float64        `firestore:"ExecutionTime" json:"execution_time,omitempty"`
	LoggedinUsers               []string       `firestore:"LoggedinUsers" json:"loggedin_users,omitempty"`
	CurrentSessions             []string       `firestore:"CurrentSessions" json:"current_sessions,omitempty"`
	Decision                    string         `firestore:"Decision" json:"decision"`
	FileBundleID                string         `firestore:"FileBundleID" json:"file_bundle_id,omitempty"`
	FileBundlePath              string         `firestore:"FileBundlePath" json:"file_bundle_path,omitempty"`
	FileBundleExecutableRelPath string         `firestore:"FileBundleExecutableRelPath" json:"file_bundle_executable_rel_path,omitempty"`
	FileBundleName              string         `firestore:"FileBundleName" json:"file_bundle_name,omitempty"`
	FileBundleVersion           string         `firestore:"FileBundleVersion" json:"file_bundle_version,omitempty"`
	FileBundleVersionString     string         `firestore:"FileBundleVersionString" json:"file_bundle_version_string,omitempty"`
	FileBundleHash              string         `firestore:"FileBundleHash" json:"file_bundle_hash,omitempty"`
	FileBundleHashMillis        float64        `firestore:"FileBundleHashMillis" json:"file_bundle_hash_millis,omitempty"`
	FileBundleBinaryCount       int            `firestore:"FileBundleBinaryCount" json:"file_bundle_binary_count,omitempty"`
	PID                         int            `firestore:"PID" json:"pid,omitempty"`
	PPID                        int            `firestore:"PPID" json:"ppid,omitempty"`
	ParentName                  string         `firestore:"ParentName" json:"parent_name,omitempty"`
	QuarantineDataURL           string         `firestore:"QuarantineDataURL" json:"quarantine_data_url,omitempty"`
	QuarantineRefererURL        string         `firestore:"QuarantineRefererURL" json:"quarantine_referer_url,omitempty"`
	QuarantineTimestamp         float64        `firestore:"QuarantineTimestamp" json:"quarantine_timestamp,omitempty"`
	QuarantineAgentBundleID     string         `firestore:"QuarantineAgentBundleID" json:"quarantine_agent_bundle_id,omitempty"`
	SigningChain                []SigningChain `firestore:"SigningChain" json:"signing_chain,omitempty"`
	SigningID                   string         `firestore:"SigningID" json:"signing_id,omitempty"`
	TeamID                      string         `firestore:"TeamID" json:"team_id,omitempty"`
	CDHash                      string         `firestore:"CDHash" json:"cdhash,omitempty"`
	VirusTotalResult            int            `firestore:"VirusTotalResult" json:"virustotalresult,omitempty"`
	LastUpdated                 time.Time      `firestore:"LastUpdated,omitempty"`
	Severity                    string         `json:"severity,omitempty"`                      // For GCP logging output
	Labels                      *Labels        `json:"logging.googleapis.com/labels,omitempty"` // For GCP logging output
}

type SigningChain struct {
	Sha256     string `json:"sha256"`
	CN         string `json:"cn"`
	Org        string `json:"org"`
	OU         string `json:"ou"`
	ValidFrom  int    `json:"valid_from"`
	ValidUntil int    `json:"valid_until"`
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
