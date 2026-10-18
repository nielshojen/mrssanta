package postflight

import (
	"testing"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	testNow          = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	testSyncStarted  = primitive.NewDateTimeFromTime(testNow.Add(-2 * time.Minute))
	testPrevLastSync = primitive.NewDateTimeFromTime(testNow.Add(-24 * time.Hour))
	testPrevClean    = primitive.NewDateTimeFromTime(testNow.Add(-240 * time.Hour))
)

func TestCompleteSyncNormalAdvancesLastSyncToSyncStart(t *testing.T) {
	device := &Device{
		PendingSyncType: "NORMAL",
		SyncStartedAt:   testSyncStarted,
		LastSync:        testPrevLastSync,
		LastCleanSync:   testPrevClean,
	}

	ok := completeSync(device, Request{RulesReceived: 3, RulesProcessed: 3}, testNow)

	assert.True(t, ok)
	assert.Equal(t, testSyncStarted, device.LastSync)
	assert.Equal(t, testPrevClean, device.LastCleanSync, "a normal sync must not count as a clean sync")
	assert.Empty(t, device.PendingSyncType)
}

func TestCompleteSyncWithNoRulesStillAdvancesLastSync(t *testing.T) {
	device := &Device{
		PendingSyncType: "NORMAL",
		SyncStartedAt:   testSyncStarted,
		LastSync:        testPrevLastSync,
	}

	ok := completeSync(device, Request{}, testNow)

	assert.True(t, ok)
	assert.Equal(t, testSyncStarted, device.LastSync)
}

func TestCompleteSyncCleanAllRecordsLastCleanSync(t *testing.T) {
	device := &Device{
		PendingSyncType:  "CLEAN_ALL",
		SyncStartedAt:    testSyncStarted,
		LastCleanSync:    testPrevClean,
		NeedsCleanSync:   true,
		RequestCleanSync: true,
	}

	ok := completeSync(device, Request{RulesReceived: 1200, RulesProcessed: 1200}, testNow)

	assert.True(t, ok)
	assert.Equal(t, testSyncStarted, device.LastSync)
	assert.Equal(t, testSyncStarted, device.LastCleanSync)
	assert.False(t, device.NeedsCleanSync)
	assert.False(t, device.RequestCleanSync)
	assert.Empty(t, device.PendingSyncType)
}

func TestCompleteSyncCleanDoesNotRecordLastCleanSync(t *testing.T) {
	device := &Device{
		PendingSyncType:  "CLEAN",
		SyncStartedAt:    testSyncStarted,
		LastCleanSync:    testPrevClean,
		RequestCleanSync: true,
	}

	ok := completeSync(device, Request{RulesReceived: 10, RulesProcessed: 10}, testNow)

	assert.True(t, ok)
	assert.Equal(t, testPrevClean, device.LastCleanSync)
	assert.False(t, device.RequestCleanSync)
}

func TestCompleteSyncWithoutSyncStartFallsBackToNow(t *testing.T) {
	device := &Device{PendingSyncType: "NORMAL"}

	ok := completeSync(device, Request{}, testNow)

	assert.True(t, ok)
	assert.Equal(t, primitive.NewDateTimeFromTime(testNow), device.LastSync)
}

func TestCompleteSyncRuleMismatchFlagsCleanSync(t *testing.T) {
	device := &Device{
		PendingSyncType: "CLEAN_ALL",
		SyncStartedAt:   testSyncStarted,
		LastSync:        testPrevLastSync,
		LastCleanSync:   testPrevClean,
	}

	ok := completeSync(device, Request{RulesReceived: 500, RulesProcessed: 10}, testNow)

	assert.False(t, ok)
	assert.True(t, device.NeedsCleanSync)
	assert.Equal(t, testPrevLastSync, device.LastSync, "a failed sync must not advance last_sync")
	assert.Equal(t, testPrevClean, device.LastCleanSync, "a failed clean sync must not count as completed")
	assert.Empty(t, device.PendingSyncType)
}

func TestCompleteSyncClearsAdminRequestedSync(t *testing.T) {
	for _, syncType := range []string{"CLEAN_ALL", "CLEAN_RULES", "CLEAN_STANDALONE", "CLEAN"} {
		t.Run(syncType, func(t *testing.T) {
			device := &Device{PendingSyncType: syncType, SyncStartedAt: testSyncStarted, RequestedSyncType: "CLEAN_RULES"}

			assert.True(t, completeSync(device, Request{}, testNow))
			assert.Empty(t, device.RequestedSyncType)
		})
	}
}

func TestCompleteSyncKeepsAdminRequestAfterNormalOrFailedSync(t *testing.T) {
	normal := &Device{PendingSyncType: "NORMAL", SyncStartedAt: testSyncStarted, RequestedSyncType: "CLEAN_RULES"}
	assert.True(t, completeSync(normal, Request{}, testNow))
	assert.Equal(t, "CLEAN_RULES", normal.RequestedSyncType, "a sync that started before the request must not satisfy it")

	failed := &Device{PendingSyncType: "CLEAN_RULES", SyncStartedAt: testSyncStarted, RequestedSyncType: "CLEAN_RULES"}
	assert.False(t, completeSync(failed, Request{RulesReceived: 10}, testNow))
	assert.Equal(t, "CLEAN_RULES", failed.RequestedSyncType)
}

func TestRequestFromProto(t *testing.T) {
	req := requestFromProto(syncv1.PostflightRequest_builder{
		RulesReceived:  12,
		RulesProcessed: 11,
		MachineId:      "machine-1",
		SyncType:       syncv1.SyncType_CLEAN_RULES,
		RulesHash:      "abc",
	}.Build())

	assert.Equal(t, Request{
		RulesReceived:  12,
		RulesProcessed: 11,
		MachineID:      "machine-1",
		SyncType:       "CLEAN_RULES",
		RulesHash:      "abc",
	}, req)

	assert.Empty(t, requestFromProto(&syncv1.PostflightRequest{}).SyncType)
}

func TestRequestFromSantaJSON(t *testing.T) {
	// Santa sends sync_type and rules_hash without a json_name override, so
	// they arrive as lowerCamelCase.
	var msg syncv1.PostflightRequest
	require.NoError(t, protojson.Unmarshal(
		[]byte(`{"rules_received":3,"rules_processed":3,"machine_id":"m1","syncType":"CLEAN_ALL","rulesHash":"h"}`), &msg))

	req := requestFromProto(&msg)
	assert.Equal(t, "CLEAN_ALL", req.SyncType)
	assert.Equal(t, "h", req.RulesHash)
	assert.Equal(t, uint32(3), req.RulesProcessed)
}

func TestCompleteSyncAllowsDiffUpToCompilerRuleCount(t *testing.T) {
	device := &Device{
		PendingSyncType:   "NORMAL",
		SyncStartedAt:     testSyncStarted,
		CompilerRuleCount: 2,
	}

	ok := completeSync(device, Request{RulesReceived: 12, RulesProcessed: 10}, testNow)

	assert.True(t, ok)
	assert.False(t, device.NeedsCleanSync)
	assert.Equal(t, testSyncStarted, device.LastSync)
}
