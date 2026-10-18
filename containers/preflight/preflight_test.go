package preflight

import (
	"testing"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const testMaxAge = 30 * 24 * time.Hour

func TestDetermineSyncType(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	recent := primitive.NewDateTimeFromTime(now.Add(-24 * time.Hour))
	stale := primitive.NewDateTimeFromTime(now.Add(-testMaxAge - time.Hour))

	tests := []struct {
		name     string
		existing *Device
		incoming *Device
		want     string
	}{
		{
			name:     "new device gets clean all",
			existing: nil,
			incoming: &Device{},
			want:     "CLEAN_ALL",
		},
		{
			name:     "device without recorded clean sync gets clean all",
			existing: &Device{},
			incoming: &Device{},
			want:     "CLEAN_ALL",
		},
		{
			name:     "stale clean sync gets clean all",
			existing: &Device{LastCleanSync: stale},
			incoming: &Device{},
			want:     "CLEAN_ALL",
		},
		{
			name:     "stored needs_clean_sync gets clean all",
			existing: &Device{LastCleanSync: recent, NeedsCleanSync: true},
			incoming: &Device{},
			want:     "CLEAN_ALL",
		},
		{
			name:     "incoming needs_clean_sync gets clean all",
			existing: &Device{LastCleanSync: recent},
			incoming: &Device{NeedsCleanSync: true},
			want:     "CLEAN_ALL",
		},
		{
			name:     "client requested clean sync gets clean",
			existing: &Device{LastCleanSync: recent},
			incoming: &Device{RequestCleanSync: true},
			want:     "CLEAN",
		},
		{
			name:     "clean all takes precedence over client requested clean",
			existing: &Device{LastCleanSync: stale},
			incoming: &Device{RequestCleanSync: true},
			want:     "CLEAN_ALL",
		},
		{
			name:     "up to date device gets normal",
			existing: &Device{LastCleanSync: recent},
			incoming: &Device{},
			want:     "NORMAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := determineSyncType(tt.existing, tt.incoming, now, testMaxAge)
			assert.Equal(t, tt.want, got)
			assert.NotEmpty(t, reason)
		})
	}
}

func TestDetermineSyncTypeDoesNotModifyDevices(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	stale := primitive.NewDateTimeFromTime(now.Add(-testMaxAge - time.Hour))
	existing := &Device{LastCleanSync: stale}
	incoming := &Device{}

	syncType, _ := determineSyncType(existing, incoming, now, testMaxAge)

	assert.Equal(t, "CLEAN_ALL", syncType)
	assert.Equal(t, stale, existing.LastCleanSync, "last_clean_sync must only be advanced by postflight")
	assert.Zero(t, incoming.LastCleanSync)
}

func TestDetermineSyncTypeDetectsMissingCleanSync(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

	syncType, reason := determineSyncType(&Device{}, &Device{}, now, testMaxAge)

	assert.Equal(t, "CLEAN_ALL", syncType)
	assert.Equal(t, "no recorded clean sync", reason)
}

func TestDetermineSyncTypeAdminRequest(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	recent := primitive.NewDateTimeFromTime(now.Add(-24 * time.Hour))
	stale := primitive.NewDateTimeFromTime(now.Add(-testMaxAge - time.Hour))

	syncType, reason := determineSyncType(&Device{LastCleanSync: recent, RequestedSyncType: "CLEAN_STANDALONE"}, &Device{}, now, testMaxAge)
	assert.Equal(t, "CLEAN_STANDALONE", syncType)
	assert.Equal(t, "requested by admin", reason)

	syncType, _ = determineSyncType(&Device{LastCleanSync: recent, RequestedSyncType: "clean_rules"}, &Device{RequestCleanSync: true}, now, testMaxAge)
	assert.Equal(t, "CLEAN_RULES", syncType, "admin request wins over a client requested CLEAN")

	syncType, _ = determineSyncType(&Device{LastCleanSync: stale, RequestedSyncType: "CLEAN_STANDALONE"}, &Device{}, now, testMaxAge)
	assert.Equal(t, "CLEAN_ALL", syncType, "server forced CLEAN_ALL wins over an admin request")

	syncType, _ = determineSyncType(&Device{LastCleanSync: recent, RequestedSyncType: "NORMAL"}, &Device{}, now, testMaxAge)
	assert.Equal(t, "NORMAL", syncType, "unknown requested types are ignored")
}

func TestDeviceFromPreflight(t *testing.T) {
	req := syncv1.PreflightRequest_builder{
		SerialNumber:         "C02ABC",
		Hostname:             "mac.example.com",
		OsVersion:            "15.1",
		OsBuild:              "24B83",
		ModelIdentifier:      "Mac15,7",
		SantaVersion:         "2026.9",
		PrimaryUser:          "jdoe",
		PrimaryUserGroups:    []string{"engineering"},
		ClientMode:           syncv1.ClientMode_MONITOR,
		RequestCleanSync:     true,
		BinaryRuleCount:      1,
		CertificateRuleCount: 2,
		CompilerRuleCount:    3,
		TransitiveRuleCount:  4,
		TeamidRuleCount:      5,
		SigningidRuleCount:   6,
		CdhashRuleCount:      7,
		SipStatus:            127,
		RulesHash:            "abc",
	}.Build()

	device := deviceFromPreflight(req, "machine-1")

	assert.Equal(t, "machine-1", device.Identifier)
	assert.Equal(t, "machine-1", device.MachineID, "falls back to the URL machine ID")
	assert.Equal(t, "C02ABC", device.SerialNumber)
	assert.Equal(t, "mac.example.com", device.Hostname)
	assert.Equal(t, "15.1", device.OSVersion)
	assert.Equal(t, "24B83", device.OSBuild)
	assert.Equal(t, "Mac15,7", device.ModelIdentifier)
	assert.Equal(t, "2026.9", device.SantaVersion)
	assert.Equal(t, "jdoe", device.PrimaryUser)
	assert.Equal(t, []string{"engineering"}, device.PrimaryUserGroups)
	assert.Equal(t, clientModeMonitor, device.ReportedClientMode)
	assert.Zero(t, device.ClientMode, "the assigned mode is server-owned")
	assert.True(t, device.RequestCleanSync)
	assert.Equal(t, []uint32{1, 2, 3, 4, 5, 6, 7}, []uint32{
		device.BinaryRuleCount, device.CertificateRuleCount, device.CompilerRuleCount,
		device.TransitiveRuleCount, device.TeamIDRuleCount, device.SigningIDRuleCount, device.CDHashRuleCount,
	})
	assert.Equal(t, uint32(127), device.SIPStatus)
	assert.Equal(t, "abc", device.RulesHash)
	assert.Nil(t, device.Settings)
	assert.Empty(t, device.RequestedSyncType)
}

func TestDetermineSyncTypeHonoursMaxAge(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	tenDaysAgo := &Device{LastCleanSync: primitive.NewDateTimeFromTime(now.Add(-10 * 24 * time.Hour))}

	syncType, _ := determineSyncType(tenDaysAgo, &Device{}, now, 7*24*time.Hour)
	assert.Equal(t, "CLEAN_ALL", syncType, "10 days exceeds a 7 day max age")

	syncType, _ = determineSyncType(tenDaysAgo, &Device{}, now, 14*24*time.Hour)
	assert.Equal(t, "NORMAL", syncType, "10 days is within a 14 day max age")
}
