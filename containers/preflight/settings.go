package preflight

import (
	"context"
	"errors"
	"fmt"
	"strings"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// globalSettingsID is the _id of the global settings document in the
// "settings" collection.
const globalSettingsID = "global"

// Settings are the preflight options an admin can set globally and override
// per device. A nil field means "not set": a device override falls back to
// the global value, and the global value falls back to the built-in default
// or to Santa's own default (the field is then left out of the response).
type Settings struct {
	// DefaultClientMode is the mode assigned to newly enrolled devices:
	// MONITOR, LOCKDOWN or STANDALONE. Only read from the global settings.
	DefaultClientMode string `bson:"default_client_mode,omitempty"`

	EventBatchSize            *uint32 `bson:"event_batch_size,omitempty"`
	EnableBundles             *bool   `bson:"enable_bundles,omitempty"`
	EnableTransitiveRules     *bool   `bson:"enable_transitive_rules,omitempty"`
	EnableAllEventUpload      *bool   `bson:"enable_all_event_upload,omitempty"`
	DisableUnknownEventUpload *bool   `bson:"disable_unknown_event_upload,omitempty"`
	AllowedPathRegex          *string `bson:"allowed_path_regex,omitempty"`
	BlockedPathRegex          *string `bson:"blocked_path_regex,omitempty"`
	// OverrideFileAccessAction is NONE, AUDIT_ONLY or DISABLE.
	OverrideFileAccessAction *string      `bson:"override_file_access_action,omitempty"`
	RemovableMediaPolicy     *MediaPolicy `bson:"removable_media_policy,omitempty"`
	EncryptedMediaPolicy     *MediaPolicy `bson:"encrypted_removable_media_policy,omitempty"`
	EventDetailURL           *string      `bson:"event_detail_url,omitempty"`
	EventDetailText          *string      `bson:"event_detail_text,omitempty"`
}

// MediaPolicy controls how removable media mounts are handled.
type MediaPolicy struct {
	// Action is "allow", "block" or "remount".
	Action string `bson:"action"`
	// RemountFlags are the mount flags used when Action is "remount",
	// e.g. ["rdonly", "noexec"].
	RemountFlags []string `bson:"remount_flags,omitempty"`
}

// Built-in defaults, matching the values this server has always sent.
const (
	defaultEventBatchSize        uint32 = 500
	defaultEnableBundles                = true
	defaultEnableTransitiveRules        = true
)

// mergeSettings returns override layered on top of base: every field set in
// override wins, everything else comes from base.
func mergeSettings(base, override *Settings) Settings {
	var merged Settings
	if base != nil {
		merged = *base
	}
	if override == nil {
		return merged
	}

	if override.DefaultClientMode != "" {
		merged.DefaultClientMode = override.DefaultClientMode
	}
	if override.EventBatchSize != nil {
		merged.EventBatchSize = override.EventBatchSize
	}
	if override.EnableBundles != nil {
		merged.EnableBundles = override.EnableBundles
	}
	if override.EnableTransitiveRules != nil {
		merged.EnableTransitiveRules = override.EnableTransitiveRules
	}
	if override.EnableAllEventUpload != nil {
		merged.EnableAllEventUpload = override.EnableAllEventUpload
	}
	if override.DisableUnknownEventUpload != nil {
		merged.DisableUnknownEventUpload = override.DisableUnknownEventUpload
	}
	if override.AllowedPathRegex != nil {
		merged.AllowedPathRegex = override.AllowedPathRegex
	}
	if override.BlockedPathRegex != nil {
		merged.BlockedPathRegex = override.BlockedPathRegex
	}
	if override.OverrideFileAccessAction != nil {
		merged.OverrideFileAccessAction = override.OverrideFileAccessAction
	}
	if override.RemovableMediaPolicy != nil {
		merged.RemovableMediaPolicy = override.RemovableMediaPolicy
	}
	if override.EncryptedMediaPolicy != nil {
		merged.EncryptedMediaPolicy = override.EncryptedMediaPolicy
	}
	if override.EventDetailURL != nil {
		merged.EventDetailURL = override.EventDetailURL
	}
	if override.EventDetailText != nil {
		merged.EventDetailText = override.EventDetailText
	}

	return merged
}

// defaultClientMode returns the mode for a newly enrolled device.
func (s Settings) defaultClientMode() uint32 {
	if mode, ok := clientModeFromName(s.DefaultClientMode); ok {
		return mode
	}
	return clientModeLockdown
}

// applyTo copies the settings onto a preflight response. Invalid values are
// logged and skipped rather than failing the sync.
func (s Settings) applyTo(resp *syncv1.PreflightResponse, machineID string) {
	resp.BatchSize = defaultEventBatchSize
	if s.EventBatchSize != nil && *s.EventBatchSize > 0 {
		resp.BatchSize = *s.EventBatchSize
	}

	resp.EnableBundles = boolOr(s.EnableBundles, defaultEnableBundles)
	resp.EnableTransitiveRules = boolOr(s.EnableTransitiveRules, defaultEnableTransitiveRules)
	resp.EnableAllEventUpload = s.EnableAllEventUpload
	resp.DisableUnknownEventUpload = s.DisableUnknownEventUpload
	resp.AllowedPathRegex = s.AllowedPathRegex
	resp.BlockedPathRegex = s.BlockedPathRegex
	resp.EventDetailUrl = s.EventDetailURL
	resp.EventDetailText = s.EventDetailText

	if s.OverrideFileAccessAction != nil {
		action, ok := syncv1.FileAccessAction_value[strings.ToUpper(*s.OverrideFileAccessAction)]
		if ok && action != 0 {
			resp.OverrideFileAccessAction = syncv1.FileAccessAction(action).Enum()
		} else {
			Warnf("Ignoring invalid override_file_access_action %q for %s", *s.OverrideFileAccessAction, machineID)
		}
	}

	if s.RemovableMediaPolicy != nil {
		policy, err := s.RemovableMediaPolicy.toProto()
		if err != nil {
			Warnf("Ignoring invalid removable_media_policy for %s: %v", machineID, err)
		} else {
			resp.RemovableMediaPolicy = policy
		}
	}

	if s.EncryptedMediaPolicy != nil {
		policy, err := s.EncryptedMediaPolicy.toProto()
		if err != nil {
			Warnf("Ignoring invalid encrypted_removable_media_policy for %s: %v", machineID, err)
		} else {
			resp.EncryptedRemovableMediaPolicy = policy
		}
	}
}

func (p MediaPolicy) toProto() (*syncv1.RemovableMediaPolicy, error) {
	switch strings.ToLower(strings.TrimSpace(p.Action)) {
	case "allow":
		return syncv1.RemovableMediaPolicy_builder{Allow: boolPtr(true)}.Build(), nil
	case "block":
		return syncv1.RemovableMediaPolicy_builder{Block: boolPtr(true)}.Build(), nil
	case "remount":
		if len(p.RemountFlags) == 0 {
			return nil, errors.New("remount requires at least one remount flag")
		}
		return syncv1.RemovableMediaPolicy_builder{
			Remount: syncv1.RemountPolicy_builder{Flags: p.RemountFlags}.Build(),
		}.Build(), nil
	default:
		return nil, fmt.Errorf("unknown action %q", p.Action)
	}
}

// loadGlobalSettings reads the global settings document. A missing document
// means no settings have been saved yet and all defaults apply.
func loadGlobalSettings(ctx context.Context, client *mongo.Client) (*Settings, error) {
	var settings Settings
	err := client.Database(dbName).Collection("settings").
		FindOne(ctx, bson.M{"_id": globalSettingsID}).Decode(&settings)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return &Settings{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load global settings: %w", err)
	}
	return &settings, nil
}

func boolOr(value *bool, fallback bool) *bool {
	if value != nil {
		return value
	}
	return boolPtr(fallback)
}

func boolPtr(value bool) *bool {
	return &value
}
