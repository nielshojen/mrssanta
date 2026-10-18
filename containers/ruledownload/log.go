package ruledownload

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Severity string

const (
	SeverityDebug     Severity = "DEBUG"
	SeverityInfo      Severity = "INFO"
	SeverityWarning   Severity = "WARNING"
	SeverityError     Severity = "ERROR"
	SeverityCritical  Severity = "CRITICAL"
	SeverityEmergency Severity = "EMERGENCY"
)

type logEntry struct {
	Time        string         `json:"time"`
	Severity    string         `json:"severity"`
	Level       string         `json:"level"`
	Status      string         `json:"status"`
	Message     string         `json:"message"`
	Service     string         `json:"service"`
	Component   string         `json:"component"`
	Environment string         `json:"env,omitempty"`
	Version     string         `json:"version,omitempty"`
	Fields      map[string]any `json:"fields,omitempty"`
}

var logMu sync.Mutex

// Log writes one JSON object per line. Normal logs go to stdout; real errors go
// to stderr for collectors that still rely on stream routing instead of fields.
func Debugf(format string, args ...any) {
	Logf(SeverityDebug, format, args...)
}

func Infof(format string, args ...any) {
	Logf(SeverityInfo, format, args...)
}

func Warnf(format string, args ...any) {
	Logf(SeverityWarning, format, args...)
}

func Errorf(format string, args ...any) {
	Logf(SeverityError, format, args...)
}

func Fatalf(format string, args ...any) {
	Logf(SeverityCritical, format, args...)
	os.Exit(1)
}

func Logf(severity Severity, format string, args ...any) {
	Log(severity, fmt.Sprintf(format, args...), nil)
}

func Log(severity Severity, message string, fields map[string]any) {
	severity = normalizeSeverity(severity)

	entry := logEntry{
		Time:        time.Now().UTC().Format(time.RFC3339Nano),
		Severity:    string(severity),
		Level:       strings.ToLower(string(severity)),
		Status:      logStatus(severity),
		Message:     message,
		Service:     envOrDefault("mrssanta-ruledownload", "SERVICE_NAME", "K_SERVICE", "DD_SERVICE"),
		Component:   "ruledownload",
		Environment: envOrDefault("", "ENVIRONMENT", "ENV", "DD_ENV"),
		Version:     envOrDefault("", "VERSION", "K_REVISION"),
		Fields:      fields,
	}

	line, err := json.Marshal(entry)
	if err != nil {
		line, _ = json.Marshal(logEntry{
			Time:      time.Now().UTC().Format(time.RFC3339Nano),
			Severity:  string(SeverityError),
			Level:     "error",
			Status:    "error",
			Message:   "failed to marshal log entry: " + err.Error(),
			Service:   "mrssanta-ruledownload",
			Component: "ruledownload",
		})
	}

	out := io.Writer(os.Stdout)
	if severityWritesToStderr(severity) {
		out = os.Stderr
	}

	logMu.Lock()
	defer logMu.Unlock()
	_, _ = fmt.Fprintln(out, string(line))
}

func normalizeSeverity(severity Severity) Severity {
	switch severity {
	case SeverityDebug, SeverityInfo, SeverityWarning, SeverityError, SeverityCritical, SeverityEmergency:
		return severity
	default:
		return SeverityInfo
	}
}

func severityWritesToStderr(severity Severity) bool {
	return severity == SeverityError || severity == SeverityCritical || severity == SeverityEmergency
}

func logStatus(severity Severity) string {
	if severity == SeverityWarning {
		return "warn"
	}
	return strings.ToLower(string(severity))
}

func envOrDefault(fallback string, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return fallback
}
