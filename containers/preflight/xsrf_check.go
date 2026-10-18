package preflight

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
)

// xsrfTokenHeader carries the token issued by the xsrf service.
const xsrfTokenHeader = "X-XSRF-TOKEN"

var (
	xsrfRequired bool
	xsrfSecret   []byte
)

// loadXSRFConfig reads XSRF_REQUIRED and XSRF_SECRET. Enforcement is off
// unless XSRF_REQUIRED is "true", and then the secret shared with the xsrf
// service is mandatory.
func loadXSRFConfig() error {
	xsrfRequired = strings.EqualFold(strings.TrimSpace(os.Getenv("XSRF_REQUIRED")), "true")
	xsrfSecret = []byte(os.Getenv("XSRF_SECRET"))

	if xsrfRequired && len(xsrfSecret) == 0 {
		return errors.New("XSRF_REQUIRED is true but XSRF_SECRET is not set")
	}
	return nil
}

// checkXSRF verifies the request's XSRF token when enforcement is on. On
// failure it answers 403, which makes Santa fetch a token from
// /santa/xsrf/{machine_id} and retry, and returns false.
func checkXSRF(w http.ResponseWriter, r *http.Request, machineID string) bool {
	if !xsrfRequired {
		return true
	}

	if err := verifyToken(xsrfSecret, r.Header.Get(xsrfTokenHeader), machineID, time.Now()); err != nil {
		Warnf("Rejected request from %s: %v", machineID, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": "Invalid XSRF token"}`))
		return false
	}

	return true
}
