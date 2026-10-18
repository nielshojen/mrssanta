package preflight

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setXSRF(t *testing.T, required bool, secret string) {
	t.Helper()
	oldRequired, oldSecret := xsrfRequired, xsrfSecret
	xsrfRequired, xsrfSecret = required, []byte(secret)
	t.Cleanup(func() { xsrfRequired, xsrfSecret = oldRequired, oldSecret })
}

func xsrfRequest(token string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/santa/sync/machine-1", nil)
	if token != "" {
		r.Header.Set(xsrfTokenHeader, token)
	}
	return r
}

func TestCheckXSRFDisabled(t *testing.T) {
	setXSRF(t, false, "")
	w := httptest.NewRecorder()

	assert.True(t, checkXSRF(w, xsrfRequest(""), "machine-1"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCheckXSRFAcceptsValidToken(t *testing.T) {
	setXSRF(t, true, "secret")
	w := httptest.NewRecorder()

	token := issueToken([]byte("secret"), "machine-1", time.Now())

	assert.True(t, checkXSRF(w, xsrfRequest(token), "machine-1"))
}

func TestCheckXSRFRejectsInvalidTokens(t *testing.T) {
	setXSRF(t, true, "secret")
	now := time.Now()

	tests := map[string]string{
		"missing":       "",
		"other machine": issueToken([]byte("secret"), "machine-2", now),
		"other secret":  issueToken([]byte("other"), "machine-1", now),
		"expired":       issueToken([]byte("secret"), "machine-1", now.Add(-2*tokenTTL)),
		"garbage":       "not-a-token",
	}

	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()

			assert.False(t, checkXSRF(w, xsrfRequest(token), "machine-1"))
			assert.Equal(t, http.StatusForbidden, w.Code, "Santa only fetches a token after a 403")
		})
	}
}

func TestLoadXSRFConfig(t *testing.T) {
	setXSRF(t, false, "")

	t.Setenv("XSRF_REQUIRED", "true")
	t.Setenv("XSRF_SECRET", "")
	assert.Error(t, loadXSRFConfig(), "enforcement without a secret must fail at startup")

	t.Setenv("XSRF_SECRET", "secret")
	require.NoError(t, loadXSRFConfig())
	assert.True(t, xsrfRequired)

	t.Setenv("XSRF_REQUIRED", "")
	require.NoError(t, loadXSRFConfig())
	assert.False(t, xsrfRequired)
}
