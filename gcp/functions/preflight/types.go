package preflight

import (
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Device struct {
	ID                   string             `bson:"_id,omitempty" json:"-"`
	Identifier           string             `bson:"identifier" json:"-"`
	SerialNumber         string             `bson:"serial_num" json:"serial_num"`
	Hostname             string             `bson:"hostname" json:"hostname"`
	OSVersion            string             `bson:"os_version" json:"os_version"`
	OSBuild              string             `bson:"os_build" json:"os_build"`
	ModelIdentifier      string             `bson:"model_identifier,omitempty" json:"model_identifier,omitempty"`
	SantaVersion         string             `bson:"santa_version" json:"santa_version"`
	PrimaryUser          string             `bson:"primary_user" json:"primary_user"`
	BinaryRuleCount      int                `bson:"binary_rule_count,omitempty" json:"binary_rule_count,omitempty"`
	CertificateRuleCount int                `bson:"certificate_rule_count,omitempty" json:"certificate_rule_count,omitempty"`
	CompilerRuleCount    int                `bson:"compiler_rule_count,omitempty" json:"compiler_rule_count,omitempty"`
	TransitiveRuleCount  int                `bson:"transitive_rule_count,omitempty" json:"transitive_rule_count,omitempty"`
	TeamIDRuleCount      int                `bson:"teamid_rule_count,omitempty" json:"teamid_rule_count,omitempty"`
	SigningIDRuleCount   int                `bson:"signingid_rule_count,omitempty" json:"signingid_rule_count,omitempty"`
	CDHashRuleCount      int                `bson:"cdhash_rule_count,omitempty" json:"cdhash_rule_count,omitempty"`
	ClientMode           int                `bson:"client_mode" json:"-"`
	RequestCleanSync     bool               `bson:"request_clean_sync,omitempty" json:"request_clean_sync,omitempty"`
	NeedsCleanSync       bool               `bson:"needs_clean_sync,omitempty" json:"needs_clean_sync,omitempty"`
	LastCleanSync        primitive.DateTime `bson:"last_clean_sync,omitempty" json:"last_clean_sync,omitempty"`
	LastUpdated          primitive.DateTime `bson:"last_updated,omitempty" json:"last_updated,omitempty"`
}

type Response struct {
	EnableBundles            bool   `json:"enable_bundles,omitempty"`
	EnableTransitiveRules    bool   `json:"enable_transitive_rules,omitempty"`
	BatchSize                int    `json:"batch_size,omitempty"`
	FullSyncInterval         int    `json:"full_sync_interval,omitempty"`
	ClientMode               *int   `json:"client_mode,omitempty"`
	AllowedPathRegEx         string `json:"allowed_path_regex,omitempty"`
	BlockedPathRegEx         string `json:"blocked_path_regex,omitempty"`
	BlockUSBMount            bool   `json:"block_usb_mount,omitempty"`
	RemountUSBMode           string `json:"remount_usb_mode,omitempty"`
	SyncType                 string `json:"sync_type,omitempty"`
	OverrideFileAccessAction string `json:"override_file_access_action,omitempty"`
}

func (d *Device) UnmarshalJSON(data []byte) error {
	type Alias Device
	aux := &struct {
		ClientMode string `json:"client_mode"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch aux.ClientMode {
	case "MONITOR":
		d.ClientMode = 1
	case "LOCKDOWN":
		d.ClientMode = 2
	case "STANDALONE":
		d.ClientMode = 3
	default:
		return fmt.Errorf("invalid client_mode value: %s", aux.ClientMode)
	}

	return nil
}
