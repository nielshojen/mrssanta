package api

import (
	"fmt"
	"strings"
)

// validPolicies are the rule policies Santa understands, keyed by their
// canonical name. Legacy names are mapped by legacyPolicies.
var validPolicies = map[string]bool{
	"ALLOWLIST":            true,
	"ALLOWLIST_COMPILER":   true,
	"BLOCKLIST":            true,
	"SILENT_BLOCKLIST":     true,
	"SILENT_GUI_BLOCKLIST": true,
	"SILENT_TTY_BLOCKLIST": true,
	"REMOVE":               true,
	"CEL":                  true,
}

var legacyPolicies = map[string]string{
	"WHITELIST":          "ALLOWLIST",
	"WHITELIST_COMPILER": "ALLOWLIST_COMPILER",
	"BLACKLIST":          "BLOCKLIST",
	"SILENT_BLACKLIST":   "SILENT_BLOCKLIST",
}

var validRuleTypes = map[string]bool{
	"BINARY":      true,
	"CERTIFICATE": true,
	"TEAMID":      true,
	"SIGNINGID":   true,
	"CDHASH":      true,
}

var validScopes = map[string]bool{
	"global":     true,
	"managedapp": true,
	"machine":    true,
}

// normalizeRule validates a rule submitted to the API and rewrites policy and
// rule type to their canonical upper-case names.
func normalizeRule(rule *Rule) error {
	rule.Identifier = strings.TrimSpace(rule.Identifier)
	if rule.Identifier == "" {
		return fmt.Errorf("rule is missing identifier")
	}

	policy := strings.ToUpper(strings.TrimSpace(rule.Policy))
	if mapped, ok := legacyPolicies[policy]; ok {
		policy = mapped
	}
	if policy == "" {
		return fmt.Errorf("rule with identifier %s is missing policy", rule.Identifier)
	}
	if !validPolicies[policy] {
		return fmt.Errorf("rule with identifier %s has unknown policy %q", rule.Identifier, rule.Policy)
	}
	rule.Policy = policy

	ruleType := strings.ToUpper(strings.TrimSpace(rule.RuleType))
	if ruleType == "" {
		return fmt.Errorf("rule with identifier %s is missing rule type", rule.Identifier)
	}
	if !validRuleTypes[ruleType] {
		return fmt.Errorf("rule with identifier %s has unknown rule type %q", rule.Identifier, rule.RuleType)
	}
	rule.RuleType = ruleType

	if rule.Scope != "" && !validScopes[rule.Scope] {
		return fmt.Errorf("rule with identifier %s has unknown scope %q", rule.Identifier, rule.Scope)
	}

	rule.CELExpr = strings.TrimSpace(rule.CELExpr)
	if policy == "CEL" && rule.CELExpr == "" {
		return fmt.Errorf("rule with identifier %s has policy CEL but no cel_expr", rule.Identifier)
	}

	return nil
}
