package xsrf

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testSecret = []byte("test-secret")

func TestTokenRoundTrip(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	token := issueToken(testSecret, "machine-1", now)

	assert.NoError(t, verifyToken(testSecret, token, "machine-1", now))
	assert.NoError(t, verifyToken(testSecret, token, "machine-1", now.Add(tokenTTL)), "valid until expiry")
}

func TestTokenRejections(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	token := issueToken(testSecret, "machine-1", now)
	parts := strings.Split(token, ".")

	tests := []struct {
		name      string
		token     string
		machineID string
		secret    []byte
		at        time.Time
		want      error
	}{
		{name: "other machine", token: token, machineID: "machine-2", secret: testSecret, at: now, want: errWrongMachine},
		{name: "expired", token: token, machineID: "machine-1", secret: testSecret, at: now.Add(tokenTTL + time.Second), want: errExpiredToken},
		{name: "other secret", token: token, machineID: "machine-1", secret: []byte("other"), at: now, want: errBadSignature},
		{name: "extended expiry", token: parts[0] + ".9999999999." + parts[2], machineID: "machine-1", secret: testSecret, at: now, want: errBadSignature},
		{name: "empty", token: "", machineID: "machine-1", secret: testSecret, at: now, want: errMalformedToken},
		{name: "two parts", token: parts[0] + "." + parts[1], machineID: "machine-1", secret: testSecret, at: now, want: errMalformedToken},
		{name: "bad signature encoding", token: parts[0] + "." + parts[1] + ".!!!", machineID: "machine-1", secret: testSecret, at: now, want: errMalformedToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, verifyToken(tt.secret, tt.token, tt.machineID, tt.at), tt.want)
		})
	}
}

func setupHandler(t *testing.T) {
	t.Helper()
	validAPIKey = "api-key"
	secret = testSecret
}

func TestHandlerIssuesToken(t *testing.T) {
	setupHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/santa/xsrf/machine-1", nil)
	r.Header.Set("X-API-Key", "api-key")
	w := httptest.NewRecorder()

	Handler(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	token := w.Header().Get("X-XSRF-TOKEN")
	require.NotEmpty(t, token)
	assert.NoError(t, verifyToken(testSecret, token, "machine-1", time.Now()))
}

func TestHandlerRejectsBadRequests(t *testing.T) {
	setupHandler(t)

	tests := []struct {
		name   string
		method string
		path   string
		apiKey string
		want   int
	}{
		{name: "wrong method", method: http.MethodGet, path: "/santa/xsrf/machine-1", apiKey: "api-key", want: http.StatusMethodNotAllowed},
		{name: "missing API key", method: http.MethodPost, path: "/santa/xsrf/machine-1", want: http.StatusUnauthorized},
		{name: "wrong API key", method: http.MethodPost, path: "/santa/xsrf/machine-1", apiKey: "nope", want: http.StatusUnauthorized},
		{name: "missing machine ID", method: http.MethodPost, path: "/santa/xsrf/", apiKey: "api-key", want: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.apiKey != "" {
				r.Header.Set("X-API-Key", tt.apiKey)
			}
			w := httptest.NewRecorder()

			Handler(w, r)

			assert.Equal(t, tt.want, w.Code)
			assert.Empty(t, w.Header().Get("X-XSRF-TOKEN"))
		})
	}
}

func TestMachineIDFromQuery(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/santa/xsrf/?machine_id=machine-9", nil)
	machineID, ok := machineIDFromRequest(r)
	assert.True(t, ok)
	assert.Equal(t, "machine-9", machineID)
}
