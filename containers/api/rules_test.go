package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestNormalizeRuleCanonicalisesValues(t *testing.T) {
	rule := &Rule{Identifier: " abc ", Policy: "silent_gui_blocklist", RuleType: "signingid", Scope: "global"}

	require.NoError(t, normalizeRule(rule))
	assert.Equal(t, "abc", rule.Identifier)
	assert.Equal(t, "SILENT_GUI_BLOCKLIST", rule.Policy)
	assert.Equal(t, "SIGNINGID", rule.RuleType)
}

func TestNormalizeRuleMapsLegacyPolicies(t *testing.T) {
	rule := &Rule{Identifier: "abc", Policy: "WHITELIST", RuleType: "BINARY"}

	require.NoError(t, normalizeRule(rule))
	assert.Equal(t, "ALLOWLIST", rule.Policy)
}

func TestNormalizeRuleAcceptsAllPolicies(t *testing.T) {
	for policy := range validPolicies {
		t.Run(policy, func(t *testing.T) {
			rule := &Rule{Identifier: "abc", Policy: policy, RuleType: "TEAMID", CELExpr: "true"}
			assert.NoError(t, normalizeRule(rule))
		})
	}
}

func TestNormalizeRuleRejectsInvalidRules(t *testing.T) {
	tests := []struct {
		name string
		rule Rule
	}{
		{name: "missing identifier", rule: Rule{Policy: "ALLOWLIST", RuleType: "BINARY"}},
		{name: "missing policy", rule: Rule{Identifier: "abc", RuleType: "BINARY"}},
		{name: "unknown policy", rule: Rule{Identifier: "abc", Policy: "MAYBE", RuleType: "BINARY"}},
		{name: "missing rule type", rule: Rule{Identifier: "abc", Policy: "ALLOWLIST"}},
		{name: "unknown rule type", rule: Rule{Identifier: "abc", Policy: "ALLOWLIST", RuleType: "PATH"}},
		{name: "unknown scope", rule: Rule{Identifier: "abc", Policy: "ALLOWLIST", RuleType: "BINARY", Scope: "everyone"}},
		{name: "CEL without expression", rule: Rule{Identifier: "abc", Policy: "CEL", RuleType: "TEAMID", CELExpr: "  "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, normalizeRule(&tt.rule))
		})
	}
}

func TestUnassignUpdateRecordsRemoval(t *testing.T) {
	now := bson.NewDateTimeFromTime(time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC))

	update := unassignUpdate("machine-1", now)

	assert.Equal(t, bson.M{"assigned": "machine-1"}, update["$pull"])
	assert.Equal(t, bson.M{"removed_from": "machine-1"}, update["$addToSet"])
	assert.Equal(t, bson.M{"last_updated": now}, update["$set"], "last_updated must change so incremental syncs pick up the REMOVE")
}

func TestDeleteRuleUpdateTombstones(t *testing.T) {
	now := bson.NewDateTimeFromTime(time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC))

	update := deleteRuleUpdate(now)

	assert.Equal(t, bson.M{"deleted": true, "deleted_at": now, "last_updated": now}, update["$set"])
}
