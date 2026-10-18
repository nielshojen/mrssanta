package eventupload

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
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

	defaultLogstashSource  = "mrssanta"
	defaultLogstashService = "mrssanta-eventupload"
	defaultEnvironment     = "dev"
	logstashDialTimeout    = 5 * time.Second
	logstashWriteTimeout   = 5 * time.Second
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

var (
	conn      net.Conn
	connMutex sync.Mutex
	logMu     sync.Mutex
)

func getConnectionLocked(logstashAddr string) (net.Conn, error) {
	if conn != nil {
		return conn, nil
	}

	c, err := (&net.Dialer{Timeout: logstashDialTimeout}).Dial("tcp", logstashAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to logstash: %w", err)
	}

	conn = c
	return conn, nil
}

func logstashAddress() string {
	return strings.TrimSpace(os.Getenv("LOGSTASH_TCP_ADDR"))
}

func resetConnectionLocked() {
	if conn == nil {
		return
	}

	_ = conn.Close()
	conn = nil
}

func tagPair(key, value string) string {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" || value == "" {
		return ""
	}

	value = strings.ReplaceAll(value, ",", "_")
	return key + ":" + value
}

func joinTags(pairs ...string) string {
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ",")
}

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
		Service:     envOrDefault("mrssanta-eventupload", "SERVICE_NAME", "K_SERVICE"),
		Component:   "eventupload",
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
			Service:   "mrssanta-eventupload",
			Component: "eventupload",
		})
	}

	writeJSONLine(severity, line)
}

func logEvent(event Event) error {
	severity := normalizeSeverity(Severity(strings.ToUpper(strings.TrimSpace(event.Severity))))

	localEvent := event
	localEvent.Severity = string(severity)
	localEvent.Service = ""
	localEvent.DDSource = ""
	localEvent.DDTags = ""

	logData, err := json.Marshal(localEvent)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	writeJSONLine(severity, logData)

	logstashAddr := logstashAddress()
	if logstashAddr == "" {
		Log(SeverityWarning, "Skipping Logstash ship because LOGSTASH_TCP_ADDR is empty", logstashLogFields(event, "", 0, 0))
		return nil
	}

	logstashEvent := event
	logstashEvent.Severity = string(severity)
	applyLogstashFields(&logstashEvent)

	logstashData, err := json.Marshal(logstashEvent)
	if err != nil {
		return fmt.Errorf("logstash json marshal error: %w", err)
	}

	Log(SeverityInfo, "Shipping event to Logstash", logstashLogFields(event, logstashAddr, len(logstashData), 0))

	attempts, err := writeLogstashLine(logstashData, logstashAddr)
	if err != nil {
		fields := logstashLogFields(event, logstashAddr, len(logstashData), attempts)
		fields["error"] = err.Error()
		Log(SeverityError, "Failed to ship event to Logstash", fields)
		return err
	}

	Log(SeverityInfo, "Event shipped to Logstash", logstashLogFields(event, logstashAddr, len(logstashData), attempts))
	return nil
}

func writeLogstashLine(logstashData []byte, logstashAddr string) (int, error) {
	if logstashAddr == "" {
		return 0, nil
	}

	line := append(append([]byte(nil), logstashData...), '\n')
	var lastErr error
	attempts := 0

	for attempt := 0; attempt < 2; attempt++ {
		attempts++
		connMutex.Lock()

		c, err := getConnectionLocked(logstashAddr)
		if err == nil && c != nil {
			_ = c.SetWriteDeadline(time.Now().Add(logstashWriteTimeout))
			_, err = c.Write(line)
			_ = c.SetWriteDeadline(time.Time{})
		}

		if err == nil {
			connMutex.Unlock()
			return attempts, nil
		}

		lastErr = err
		resetConnectionLocked()
		connMutex.Unlock()
	}

	return attempts, fmt.Errorf("failed to write to logstash: %w", lastErr)
}

func logstashLogFields(event Event, logstashAddr string, payloadBytes int, attempts int) map[string]any {
	fields := map[string]any{
		"logstash_addr": logstashAddr,
		"payload_bytes": payloadBytes,
	}

	if attempts > 0 {
		fields["attempts"] = attempts
	}
	if event.FileSha256 != "" {
		fields["file_sha256"] = event.FileSha256
	}
	if event.FileName != "" {
		fields["file_name"] = event.FileName
	}
	if event.Decision != "" {
		fields["decision"] = event.Decision
	}

	return fields
}

func applyLogstashFields(event *Event) {
	event.Service = ""
	event.DDSource = ""
	event.DDTags = ""

	ddSource := envOrDefault(defaultLogstashSource, "DD_SOURCE")
	ddService := envOrDefault(defaultLogstashService, "DD_SERVICE", "SERVICE_NAME", "K_SERVICE")
	env := envOrDefault(defaultEnvironment, "ENVIRONMENT", "ENV", "DD_ENV")
	owner := strings.TrimSpace(os.Getenv("OWNER"))
	team := strings.TrimSpace(os.Getenv("TEAM"))
	version := strings.TrimSpace(os.Getenv("VERSION"))
	costcenter := strings.TrimSpace(os.Getenv("COSTCENTER"))

	event.Service = ddSource
	event.DDSource = ddSource

	event.DDTags = joinTags(
		tagPair("env", env),
		tagPair("app", ddSource),
		tagPair("service", ddService),
		tagPair("source", ddSource),
		tagPair("owner", owner),
		tagPair("team", team),
		tagPair("costcenter", costcenter),
		tagPair("version", version),
	)
}

func writeJSONLine(severity Severity, line []byte) {
	out := io.Writer(os.Stdout)
	if severityWritesToStderr(normalizeSeverity(severity)) {
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
