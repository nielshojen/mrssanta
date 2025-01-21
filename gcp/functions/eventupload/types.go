package eventupload

import "time"

type Event struct {
	FileSha256                  string         `json:"file_sha256"`
	FilePath                    string         `json:"file_path"`
	FileName                    string         `json:"file_name"`
	ExecutingUser               string         `json:"executing_user,omitempty"`
	ExecutionTime               float64        `json:"execution_time,omitempty"`
	LoggedinUsers               []string       `json:"loggedin_users,omitempty"`
	CurrentSessions             []string       `json:"current_sessions,omitempty"`
	Decision                    string         `json:"decision"`
	FileBundleID                string         `json:"file_bundle_id,omitempty"`
	FileBundlePath              string         `json:"file_bundle_path,omitempty"`
	FileBundleExecutableRelPath string         `json:"file_bundle_executable_rel_path,omitempty"`
	FileBundleName              string         `json:"file_bundle_name,omitempty"`
	FileBundleVersion           string         `json:"file_bundle_version,omitempty"`
	FileBundleVersionString     string         `json:"file_bundle_version_string,omitempty"`
	FileBundleHash              string         `json:"file_bundle_hash,omitempty"`
	FileBundleHashBillis        float64        `json:"file_bundle_hash_millis,omitempty"`
	FileBundleBinaryCount       int            `json:"file_bundle_binary_count,omitempty"`
	PID                         int            `json:"pid,omitempty"`
	PPID                        int            `json:"ppid,omitempty"`
	ParentName                  string         `json:"parent_name,omitempty"`
	QuarantineDataURL           string         `json:"quarantine_data_url,omitempty"`
	QuarantineRefererURL        string         `json:"quarantine_referer_url,omitempty"`
	QuarantineTimestamp         float64        `json:"quarantine_timestamp,omitempty"`
	QuarantineAgentBundleID     string         `json:"quarantine_agent_bundle_id,omitempty"`
	SigningChain                []SigningChain `json:"signing_chain,omitempty"`
	SigningID                   string         `json:"signing_id,omitempty"`
	TeamID                      string         `json:"team_id,omitempty"`
	CDHash                      string         `json:"cdhash,omitempty"`
	VirusTotalResult            int            `firestore:"virustutalresult" json:"virustotalresult,omitempty"`
	LastUpdated                 time.Time      `firestore:"last_updated,omitempty"`
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
