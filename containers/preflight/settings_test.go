package preflight

import (
	"testing"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(value string) *string { return &value }
func u32Ptr(value uint32) *uint32 { return &value }

func TestMergeSettingsDeviceOverridesWin(t *testing.T) {
	global := &Settings{
		DefaultClientMode:    "MONITOR",
		EventBatchSize:       u32Ptr(100),
		EnableAllEventUpload: boolPtr(true),
		BlockedPathRegex:     strPtr("^/tmp/"),
		EventDetailURL:       strPtr("https://santa.example.com/%file_sha%"),
		RemovableMediaPolicy: &MediaPolicy{Action: "block"},
	}
	device := &Settings{
		EnableAllEventUpload: boolPtr(false),
		RemovableMediaPolicy: &MediaPolicy{Action: "allow"},
	}

	merged := mergeSettings(global, device)

	assert.Equal(t, "MONITOR", merged.DefaultClientMode)
	assert.Equal(t, uint32(100), *merged.EventBatchSize)
	assert.False(t, *merged.EnableAllEventUpload, "device override must win, even when false")
	assert.Equal(t, "^/tmp/", *merged.BlockedPathRegex)
	assert.Equal(t, "allow", merged.RemovableMediaPolicy.Action)
	assert.Equal(t, "https://santa.example.com/%file_sha%", *merged.EventDetailURL)
}

func TestMergeSettingsHandlesNil(t *testing.T) {
	assert.Equal(t, Settings{}, mergeSettings(nil, nil))

	global := &Settings{EventBatchSize: u32Ptr(10)}
	assert.Equal(t, *global, mergeSettings(global, nil))
}

func TestDefaultClientMode(t *testing.T) {
	assert.Equal(t, clientModeLockdown, Settings{}.defaultClientMode())
	assert.Equal(t, clientModeMonitor, Settings{DefaultClientMode: "monitor"}.defaultClientMode())
	assert.Equal(t, clientModeStandalone, Settings{DefaultClientMode: "STANDALONE"}.defaultClientMode())
	assert.Equal(t, clientModeLockdown, Settings{DefaultClientMode: "bogus"}.defaultClientMode())
	assert.Equal(t, clientModeLockdown, Settings{DefaultClientMode: "UNKNOWN_CLIENT_MODE"}.defaultClientMode())
}

func TestApplyToDefaults(t *testing.T) {
	resp := &syncv1.PreflightResponse{}
	Settings{}.applyTo(resp, "machine-1")

	assert.Equal(t, defaultEventBatchSize, resp.GetBatchSize())
	assert.True(t, resp.GetEnableBundles())
	assert.True(t, resp.GetEnableTransitiveRules())
	assert.False(t, resp.HasEnableAllEventUpload(), "unset settings must be left out so Santa keeps its default")
	assert.False(t, resp.HasRemovableMediaPolicy())
	assert.False(t, resp.HasOverrideFileAccessAction())
	assert.False(t, resp.HasEventDetailUrl())
}

func TestApplyToAllSettings(t *testing.T) {
	settings := Settings{
		EventBatchSize:            u32Ptr(25),
		EnableBundles:             boolPtr(false),
		EnableTransitiveRules:     boolPtr(false),
		EnableAllEventUpload:      boolPtr(true),
		DisableUnknownEventUpload: boolPtr(true),
		AllowedPathRegex:          strPtr("^/opt/allowed/"),
		BlockedPathRegex:          strPtr("^/tmp/"),
		OverrideFileAccessAction:  strPtr("audit_only"),
		RemovableMediaPolicy:      &MediaPolicy{Action: "remount", RemountFlags: []string{"rdonly", "noexec"}},
		EncryptedMediaPolicy:      &MediaPolicy{Action: "allow"},
		EventDetailURL:            strPtr("https://santa.example.com/blockables/%file_sha%"),
		EventDetailText:           strPtr("Request access"),
	}

	resp := &syncv1.PreflightResponse{}
	settings.applyTo(resp, "machine-1")

	assert.Equal(t, uint32(25), resp.GetBatchSize())
	assert.False(t, resp.GetEnableBundles())
	assert.False(t, resp.GetEnableTransitiveRules())
	assert.True(t, resp.GetEnableAllEventUpload())
	assert.True(t, resp.GetDisableUnknownEventUpload())
	assert.Equal(t, "^/opt/allowed/", resp.GetAllowedPathRegex())
	assert.Equal(t, "^/tmp/", resp.GetBlockedPathRegex())
	assert.Equal(t, syncv1.FileAccessAction_AUDIT_ONLY, resp.GetOverrideFileAccessAction())
	assert.Equal(t, []string{"rdonly", "noexec"}, resp.GetRemovableMediaPolicy().GetRemount().GetFlags())
	assert.True(t, resp.GetEncryptedRemovableMediaPolicy().GetAllow())
	assert.Equal(t, "https://santa.example.com/blockables/%file_sha%", resp.GetEventDetailUrl())
	assert.Equal(t, "Request access", resp.GetEventDetailText())
}

func TestApplyToSkipsInvalidValues(t *testing.T) {
	settings := Settings{
		EventBatchSize:           u32Ptr(0),
		OverrideFileAccessAction: strPtr("sometimes"),
		RemovableMediaPolicy:     &MediaPolicy{Action: "remount"},
		EncryptedMediaPolicy:     &MediaPolicy{Action: "explode"},
	}

	resp := &syncv1.PreflightResponse{}
	settings.applyTo(resp, "machine-1")

	assert.Equal(t, defaultEventBatchSize, resp.GetBatchSize())
	assert.False(t, resp.HasOverrideFileAccessAction())
	assert.False(t, resp.HasRemovableMediaPolicy(), "remount without flags is invalid")
	assert.False(t, resp.HasEncryptedRemovableMediaPolicy())
}

func TestMediaPolicyToProto(t *testing.T) {
	allow, err := MediaPolicy{Action: "Allow"}.toProto()
	require.NoError(t, err)
	assert.True(t, allow.GetAllow())

	block, err := MediaPolicy{Action: "block"}.toProto()
	require.NoError(t, err)
	assert.True(t, block.GetBlock())

	remount, err := MediaPolicy{Action: "remount", RemountFlags: []string{"rdonly"}}.toProto()
	require.NoError(t, err)
	assert.Equal(t, []string{"rdonly"}, remount.GetRemount().GetFlags())

	_, err = MediaPolicy{Action: ""}.toProto()
	assert.Error(t, err)
}

func TestBuildPreflightResponse(t *testing.T) {
	resp := buildPreflightResponse(Settings{}, clientModeStandalone, "CLEAN_RULES", 600, "machine-1")

	assert.Equal(t, syncv1.ClientMode_STANDALONE, resp.GetClientMode())
	assert.Equal(t, syncv1.SyncType_CLEAN_RULES, resp.GetSyncType())
	assert.Equal(t, uint32(600), resp.GetFullSyncIntervalSeconds())
	assert.Equal(t, defaultEventBatchSize, resp.GetBatchSize())
}
