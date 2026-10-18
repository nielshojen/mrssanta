package eventupload

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

// tokenTTL is how long an issued token is accepted. Santa fetches one token
// per sync and reuses it for the rest of that sync, so this only needs to
// outlast a single sync.
const tokenTTL = time.Hour

var (
	errMalformedToken = errors.New("malformed XSRF token")
	errBadSignature   = errors.New("invalid XSRF token signature")
	errWrongMachine   = errors.New("XSRF token was issued to another machine")
	errExpiredToken   = errors.New("XSRF token has expired")
)

// issueToken returns a token bound to machineID that expires after tokenTTL.
// Format: base64url(machine_id) "." expiry_unix "." base64url(hmac_sha256).
// Tokens are stateless, so any replica of any sync service sharing the
// secret can verify them.
func issueToken(secret []byte, machineID string, now time.Time) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(machineID)) +
		"." + strconv.FormatInt(now.Add(tokenTTL).Unix(), 10)
	return payload + "." + base64.RawURLEncoding.EncodeToString(sign(secret, payload))
}

// verifyToken checks that token was issued by this server for machineID and
// has not expired.
func verifyToken(secret []byte, token, machineID string, now time.Time) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return errMalformedToken
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return errMalformedToken
	}
	if !hmac.Equal(signature, sign(secret, parts[0]+"."+parts[1])) {
		return errBadSignature
	}

	tokenMachine, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return errMalformedToken
	}
	if string(tokenMachine) != machineID {
		return errWrongMachine
	}

	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return errMalformedToken
	}
	if now.Unix() > expiry {
		return errExpiredToken
	}

	return nil
}

func sign(secret []byte, payload string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}
