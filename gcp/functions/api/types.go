package api

import "time"

type Rule struct {
	Identifier            string    `firestore:"Identifier" json:"identifier"`
	Policy                string    `firestore:"Policy" json:"policy"`
	RuleType              string    `firestore:"RuleType" json:"rule_type,omitempty"`
	CustomMessage         string    `firestore:"CustomMessage,omitempty" json:"custom_msg,omitempty"`
	CustomURL             string    `firestore:"CustomURL,omitempty" json:"custom_url,omitempty"`
	CreationTime          time.Time `firestore:"CreationTime,serverTimestamp,omitempty" json:"creation_time,omitempty"`
	FileBundleBinaryCount string    `firestore:"FileBundleBinaryCount,omitempty" json:"file_bundle_binary_count,omitempty"`
	FileBundleHash        string    `firestore:"FileBundleHash,omitempty" json:"file_bundle_hash,omitempty"`
	Scope                 string    `firestore:"Scope" json:"scope"`
	Assigned              []string  `firestore:"Assigned,omitempty" json:"assigned,omitempty"`
	LastUpdated           time.Time `firestore:"LastUpdated,omitempty" json:"last_updated,omitempty"`
}

type Device struct {
	Identifier           string    `firestore:"Identifier" json:"identifier"`
	SerialNumber         string    `firestore:"SerialNumber" json:"serial_num"`
	Hostname             string    `firestore:"Hostname" json:"hostname"`
	OSVersion            string    `firestore:"OSVersion" json:"os_version"`
	OSBuild              string    `firestore:"OSBuild" json:"os_build"`
	ModelIdentifier      string    `firestore:"ModelIdentifier,omitempty" json:"model_identifier,omitempty"`
	SantaVersion         string    `firestore:"SantaVersion" json:"santa_version"`
	PrimaryUser          string    `firestore:"PrimaryUser,omitempty" json:"primary_user,omitempty"`
	BinaryRuleCount      int       `firestore:"BinaryRuleCount,omitempty" json:"binary_rule_count,omitempty"`
	CertificateRuleCount int       `firestore:"CertificateRuleCount,omitempty" json:"certificate_rule_count,omitempty"`
	CompilerRuleCount    int       `firestore:"CompilerRuleCount,omitempty" json:"compiler_rule_count,omitempty"`
	TransitiveRuleCount  int       `firestore:"TransitiveRuleCount,omitempty" json:"transitive_rule_count,omitempty"`
	TeamIDRuleCount      int       `firestore:"TeamIDRuleCount,omitempty" json:"teamid_rule_count,omitempty"`
	SigningIDRuleCount   int       `firestore:"SigningIDRuleCount,omitempty" json:"signingid_rule_count,omitempty"`
	CDHashRuleCount      int       `firestore:"CDHashRuleCount,omitempty" json:"cdhash_rule_count,omitempty"`
	ClientMode           int       `firestore:"ClientMode" json:"-"`
	RequestCleanSync     bool      `firestore:"RequestCleanSync,omitempty" json:"request_clean_sync,omitempty"`
	SyncCursor           string    `firestore:"SyncCursor,omitempty" json:"sync_cursor,omitempty"`
	SyncPage             int       `firestore:"SyncPage,omitempty" json:"sync_page,omitempty"`
	LastCleanSync        time.Time `firestore:"LastCleanSync,omitempty" json:"last_clean_sync,omitempty"`
	LastUpdated          time.Time `firestore:"LastUpdated,omitempty" json:"last_updated,omitempty"`
}

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
}

type SigningChain struct {
	Sha256     string `json:"sha256"`
	CN         string `json:"cn"`
	Org        string `json:"org"`
	OU         string `json:"ou"`
	ValidFrom  int    `json:"valid_from"`
	ValidUntil int    `json:"valid_until"`
}
