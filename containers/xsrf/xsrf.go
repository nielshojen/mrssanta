package xsrf

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// tokenHeader is the response header Santa reads the token from. Santa sends
// it back in the same header on every following request of the sync.
const tokenHeader = "X-XSRF-TOKEN"

// Handler issues an XSRF token. Santa calls POST /santa/xsrf/{machine_id}
// after a sync request is answered with 403, then retries that request with
// the token in the X-XSRF-TOKEN header.
func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("X-API-Key") != validAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Unauthorized"}`)
		return
	}

	machineID, ok := machineIDFromRequest(r)
	if !ok {
		http.Error(w, "Machine ID is missing in the request URL", http.StatusBadRequest)
		return
	}

	w.Header().Set(tokenHeader, issueToken(secret, machineID, time.Now()))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{}`)

	Infof("Issued XSRF token for %s", machineID)
}

func machineIDFromRequest(r *http.Request) (string, bool) {
	if machineID := strings.TrimSpace(r.URL.Query().Get("machine_id")); machineID != "" {
		return machineID, true
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "santa" || parts[1] != "xsrf" || parts[2] == "" {
		return "", false
	}

	return parts[2], true
}
