package eventupload

import (
	"testing"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/encoding/protojson"
)

// santaEventJSON is an event as Santa's protobuf JSON printer sends it:
// fields with an explicit json_name keep it, the rest are lowerCamelCase.
const santaEventJSON = `{
	"events": [{
		"file_sha256": "abc",
		"file_path": "/Applications/Foo.app/Contents/MacOS",
		"file_name": "Foo",
		"executing_user": "jdoe",
		"logged_in_users": ["jdoe"],
		"current_sessions": ["jdoe@console"],
		"decision": "BLOCK_BINARY_MISMATCH",
		"pid": 42,
		"ppid": 1,
		"team_id": "EQHXZ8M8AV",
		"signing_id": "EQHXZ8M8AV:com.example.foo",
		"cdhash": "cd",
		"signing_chain": [{"sha256": "leaf", "cn": "Developer ID", "valid_from": 10, "valid_until": 20}],
		"entitlementInfo": {"entitlementsFiltered": true, "entitlements": [{"key": "com.apple.security.get-task-allow", "value": "true"}]},
		"csFlags": 570522385,
		"signingStatus": "SIGNING_STATUS_PRODUCTION",
		"secureSigningTime": 1700000000,
		"signingTime": 1690000000,
		"static_rule": true,
		"identity_unverified": true
	}],
	"file_access_events": [{
		"rule_name": "ProtectSSH",
		"rule_version": "v2",
		"target": "/Users/jdoe/.ssh/id_ed25519",
		"access_time": 1700000123.5,
		"decision": "FILE_ACCESS_DECISION_DENIED",
		"process_chain": [
			{"file_path": "/usr/bin/cat", "pid": 100, "signing_id": "platform:com.apple.cat"},
			{"file_path": "/bin/zsh", "pid": 99}
		]
	}],
	"audit_events": [
		{"standalone_mode_rule_creation": {"decision": "ALLOW_BINARY", "identifier": "abc", "timestamp": 1700000000}},
		{}
	],
	"machine_id": "machine-1"
}`

func decodeSantaUpload(t *testing.T) *syncv1.EventUploadRequest {
	var req syncv1.EventUploadRequest
	require.NoError(t, protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal([]byte(santaEventJSON), &req))
	return &req
}

func TestEventFromProto(t *testing.T) {
	event := eventFromProto(decodeSantaUpload(t).GetEvents()[0])

	assert.Equal(t, "abc", event.FileSha256)
	assert.Equal(t, "Foo", event.FileName)
	assert.Equal(t, []string{"jdoe"}, event.LoggedinUsers, "logged_in_users was previously dropped")
	assert.Equal(t, "BLOCK_BINARY_MISMATCH", event.Decision)
	assert.Equal(t, 42, event.PID)
	assert.Equal(t, "EQHXZ8M8AV", event.TeamID)
	assert.Equal(t, int64(570522385), event.CSFlags)
	assert.Equal(t, "SIGNING_STATUS_PRODUCTION", event.SigningStatus)
	assert.Equal(t, uint32(1700000000), event.SecureSigningTime)
	assert.Equal(t, uint32(1690000000), event.SigningTime)
	assert.True(t, event.StaticRule)
	assert.True(t, event.IdentityUnverified)
	require.Len(t, event.SigningChain, 1)
	assert.Equal(t, SigningChain{Sha256: "leaf", CN: "Developer ID", ValidFrom: 10, ValidUntil: 20}, event.SigningChain[0])
	require.NotNil(t, event.EntitlementInfo)
	assert.True(t, event.EntitlementInfo.EntitlementsFiltered)
	assert.Equal(t, []Entitlement{{Key: "com.apple.security.get-task-allow", Value: "true"}}, event.EntitlementInfo.Entitlements)
}

func TestEventFromProtoLeavesUnsetFieldsEmpty(t *testing.T) {
	event := eventFromProto(syncv1.Event_builder{FileSha256: "abc", Decision: syncv1.Decision_ALLOW_PLATFORM}.Build())

	assert.Equal(t, "ALLOW_PLATFORM", event.Decision)
	assert.Empty(t, event.SigningStatus)
	assert.Nil(t, event.EntitlementInfo)
	assert.Nil(t, event.SigningChain)
}

func TestFileAccessEventFromProto(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	event := fileAccessEventFromProto(decodeSantaUpload(t).GetFileAccessEvents()[0], "machine-1", now)

	assert.Equal(t, "machine-1", event.MachineID)
	assert.Equal(t, "ProtectSSH", event.RuleName)
	assert.Equal(t, "v2", event.RuleVersion)
	assert.Equal(t, "/Users/jdoe/.ssh/id_ed25519", event.Target)
	assert.Equal(t, "FILE_ACCESS_DECISION_DENIED", event.Decision)
	assert.InDelta(t, 1700000123.5, event.AccessTime, 0.001)
	assert.Equal(t, primitive.NewDateTimeFromTime(now), event.CreationTime)
	require.Len(t, event.ProcessChain, 2)
	assert.Equal(t, Process{FilePath: "/usr/bin/cat", PID: 100, SigningID: "platform:com.apple.cat"}, event.ProcessChain[0])
	assert.Equal(t, "/bin/zsh", event.ProcessChain[1].FilePath)
}

func TestStandaloneApprovalsFromProto(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	approvals := standaloneApprovalsFromProto(decodeSantaUpload(t).GetAuditEvents(), "machine-1", now)

	require.Len(t, approvals, 1, "audit events without a known payload are skipped")
	assert.Equal(t, StandaloneApproval{
		MachineID:    "machine-1",
		Identifier:   "abc",
		Decision:     "ALLOW_BINARY",
		Timestamp:    primitive.NewDateTimeFromTime(time.Unix(1700000000, 0)),
		CreationTime: primitive.NewDateTimeFromTime(now),
	}, approvals[0])
}

func TestStandaloneApprovalsFromProtoEmpty(t *testing.T) {
	assert.Empty(t, standaloneApprovalsFromProto(nil, "machine-1", time.Now()))
}

func TestSanitizeEventClearsUserData(t *testing.T) {
	event := eventFromProto(decodeSantaUpload(t).GetEvents()[0])
	sanitized := sanitizeEvent(event)

	assert.Empty(t, sanitized.ExecutingUser)
	assert.Nil(t, sanitized.LoggedinUsers)
	assert.Nil(t, sanitized.CurrentSessions)
	assert.Zero(t, sanitized.PID)
	assert.Equal(t, "abc", sanitized.FileSha256)
}
