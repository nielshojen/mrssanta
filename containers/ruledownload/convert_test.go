package ruledownload

import (
	"encoding/json"
	"testing"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRuleToProtoCopiesProtocolFields(t *testing.T) {
	rule := &Rule{
		Identifier:             "EQHXZ8M8AV",
		Policy:                 "BLOCKLIST",
		RuleType:               "TEAMID",
		CustomMessage:          "Not allowed",
		CustomURL:              "https://example.com/why",
		NotificationAppName:    "Example App",
		EventDetailButtonLabel: "Ask IT",
		Scope:                  "global",
	}

	got, err := rule.toProto("machine-1")
	require.NoError(t, err)

	assert.Equal(t, "EQHXZ8M8AV", got.GetIdentifier())
	assert.Equal(t, syncv1.Policy_BLOCKLIST, got.GetPolicy())
	assert.Equal(t, syncv1.RuleType_TEAMID, got.GetRuleType())
	assert.Equal(t, "Not allowed", got.GetCustomMsg())
	assert.Equal(t, "https://example.com/why", got.GetCustomUrl())
	assert.Equal(t, "Example App", got.GetNotificationAppName())
	assert.Equal(t, "Ask IT", got.GetEventDetailButtonLabel())
	assert.Empty(t, got.GetCelExpr())
}

func TestRuleToProtoDoesNotLeakInternalFields(t *testing.T) {
	rule := &Rule{
		Identifier: "abc",
		Policy:     "ALLOWLIST",
		RuleType:   "BINARY",
		Scope:      "machine",
		Assigned:   []string{"machine-1", "machine-2", "machine-3"},
		Converted:  true,
	}

	got, err := rule.toProto("machine-1")
	require.NoError(t, err)

	data, err := protojson.Marshal(got)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "machine-2")
	assert.NotContains(t, string(data), "assigned")
	assert.NotContains(t, string(data), "scope")
}

func TestRuleToProtoNewPolicies(t *testing.T) {
	for _, policy := range []string{"SILENT_GUI_BLOCKLIST", "SILENT_TTY_BLOCKLIST", "ALLOWLIST_COMPILER", "SILENT_BLOCKLIST"} {
		t.Run(policy, func(t *testing.T) {
			got, err := (&Rule{Identifier: "abc", Policy: policy, RuleType: "BINARY", Scope: "global"}).toProto("m")
			require.NoError(t, err)
			assert.Equal(t, policy, got.GetPolicy().String())
		})
	}
}

func TestRuleToProtoCEL(t *testing.T) {
	rule := &Rule{
		Identifier: "EQHXZ8M8AV",
		Policy:     "CEL",
		RuleType:   "TEAMID",
		CELExpr:    "target.signing_time >= timestamp('2025-01-01T00:00:00Z')",
		Scope:      "global",
	}

	got, err := rule.toProto("machine-1")
	require.NoError(t, err)
	assert.Equal(t, syncv1.Policy_CEL, got.GetPolicy())
	assert.Equal(t, rule.CELExpr, got.GetCelExpr())

	rule.CELExpr = "  "
	_, err = rule.toProto("machine-1")
	assert.Error(t, err, "a CEL rule without an expression is invalid")
}

func TestRuleToProtoDropsCELExprForOtherPolicies(t *testing.T) {
	got, err := (&Rule{Identifier: "abc", Policy: "ALLOWLIST", RuleType: "BINARY", Scope: "global", CELExpr: "true"}).toProto("m")
	require.NoError(t, err)
	assert.Empty(t, got.GetCelExpr())
}

func TestRuleToProtoLegacyPolicyNames(t *testing.T) {
	got, err := (&Rule{Identifier: "abc", Policy: "WHITELIST", RuleType: "BINARY", Scope: "global"}).toProto("m")
	require.NoError(t, err)
	assert.Equal(t, syncv1.Policy_ALLOWLIST, got.GetPolicy())
}

func TestRuleToProtoRejectsUnknownValues(t *testing.T) {
	_, err := (&Rule{Identifier: "abc", Policy: "ALLOWLIST", RuleType: "PATH", Scope: "global"}).toProto("m")
	assert.Error(t, err)

	_, err = (&Rule{Identifier: "abc", Policy: "MAYBE", RuleType: "BINARY", Scope: "global"}).toProto("m")
	assert.Error(t, err)

	_, err = (&Rule{Identifier: "abc", Policy: "ALLOWLIST", RuleType: "", Scope: "global"}).toProto("m")
	assert.Error(t, err)
}

func TestRuleToProtoTombstones(t *testing.T) {
	tests := []struct {
		name string
		rule Rule
	}{
		{
			name: "deleted global rule",
			rule: Rule{Identifier: "abc", Policy: "ALLOWLIST", RuleType: "BINARY", Scope: "global", Deleted: true},
		},
		{
			name: "deleted assigned rule",
			rule: Rule{Identifier: "abc", Policy: "ALLOWLIST", RuleType: "BINARY", Scope: "managedapp", Assigned: []string{"machine-1"}, Deleted: true},
		},
		{
			name: "machine unassigned from rule",
			rule: Rule{Identifier: "abc", Policy: "BLOCKLIST", RuleType: "SIGNINGID", Scope: "machine", Assigned: []string{"machine-2"}, RemovedFrom: []string{"machine-1"}},
		},
		{
			name: "unknown policy on a deleted rule still removes it",
			rule: Rule{Identifier: "abc", Policy: "MAYBE", RuleType: "BINARY", Scope: "global", Deleted: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rule.CustomMessage = "should not be sent"
			got, err := tt.rule.toProto("machine-1")
			require.NoError(t, err)
			assert.Equal(t, syncv1.Policy_REMOVE, got.GetPolicy())
			assert.Equal(t, "abc", got.GetIdentifier())
			assert.Empty(t, got.GetCustomMsg())
		})
	}
}

func TestRuleIsActiveFor(t *testing.T) {
	assert.True(t, (&Rule{Scope: "global"}).isActiveFor("m1"))
	assert.False(t, (&Rule{Scope: "global", Deleted: true}).isActiveFor("m1"))
	assert.True(t, (&Rule{Scope: "managedapp", Assigned: []string{"m1"}}).isActiveFor("m1"))
	assert.False(t, (&Rule{Scope: "managedapp", Assigned: []string{"m2"}}).isActiveFor("m1"))
	assert.False(t, (&Rule{Scope: "machine"}).isActiveFor("m1"))
}

// filterJSON renders a filter as JSON so tests can assert on its shape.
func filterJSON(t *testing.T, filter bson.M) string {
	data, err := json.Marshal(filter)
	require.NoError(t, err)
	return string(data)
}

func TestRuleDownloadFilterFullSyncExcludesTombstones(t *testing.T) {
	got := filterJSON(t, ruleDownloadFilter("machine-1", true, time.Time{}, nil))

	assert.Contains(t, got, `"deleted":{"$ne":true}`)
	assert.NotContains(t, got, "removed_from")
	assert.NotContains(t, got, "last_updated")
}

func TestRuleDownloadFilterIncrementalIncludesTombstones(t *testing.T) {
	lastSync := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	got := filterJSON(t, ruleDownloadFilter("machine-1", false, lastSync, nil))

	assert.Contains(t, got, `"removed_from":"machine-1"`)
	assert.Contains(t, got, `"last_updated":{"$gt":"2026-10-01T00:00:00Z"}`)
	assert.NotContains(t, got, `"deleted"`)
}

func TestRuleDownloadFilterCursor(t *testing.T) {
	cursor := &CursorMetadata{Identifier: "abc", LastUpdated: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC).UnixMilli()}
	got := filterJSON(t, ruleDownloadFilter("machine-1", true, time.Time{}, cursor))

	assert.Contains(t, got, `"identifier":{"$gt":"abc"}`)
	assert.Contains(t, got, `"2026-10-02T00:00:00Z"`)
}

func TestFullSyncReasonCleanRules(t *testing.T) {
	full, reason := fullSyncReason(&Device{LastSync: 1, PendingSyncType: "CLEAN_RULES"})
	assert.True(t, full)
	assert.Equal(t, "pending_sync_type=CLEAN_RULES", reason)

	full, _ = fullSyncReason(&Device{LastSync: 1, PendingSyncType: "CLEAN_STANDALONE"})
	assert.False(t, full, "CLEAN_STANDALONE only removes locally created rules")
}

func TestCursorRoundTrip(t *testing.T) {
	cursor := CursorMetadata{Identifier: "abc", LastUpdated: 1234}
	decoded, err := decodeCursor(encodeCursor(cursor))
	require.NoError(t, err)
	assert.Equal(t, cursor, decoded)

	_, err = decodeCursor("not base64!")
	assert.Error(t, err)
}
