package api

import (
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestDeviceUpdateFieldsEncodesDateFieldsAsBSONDateTime(t *testing.T) {
	var devices []*Device
	payload := []byte(`[{
		"identifier": "device-1",
		"last_sync": "2026-05-15T12:34:56Z",
		"last_clean_sync": "2026-05-14T12:34:56Z",
		"last_updated": "2026-05-13T12:34:56Z"
	}]`)

	if err := json.Unmarshal(payload, &devices); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	now := bson.NewDateTimeFromTime(time.Date(2026, 5, 15, 13, 0, 0, 0, time.UTC))
	updateFields, err := deviceUpdateFields(devices[0], now)
	if err != nil {
		t.Fatalf("deviceUpdateFields() error = %v", err)
	}

	for _, key := range []string{"last_sync", "last_clean_sync", "last_updated"} {
		if _, ok := updateFields[key].(string); ok {
			t.Fatalf("%s was encoded as string in update map", key)
		}
	}

	encoded, err := bson.Marshal(updateFields)
	if err != nil {
		t.Fatalf("bson.Marshal() error = %v", err)
	}

	raw := bson.Raw(encoded)
	for _, key := range []string{"last_sync", "last_clean_sync", "last_updated"} {
		if got := raw.Lookup(key).Type; got != bson.TypeDateTime {
			t.Fatalf("%s BSON type = %s, want %s", key, got, bson.TypeDateTime)
		}
	}
}
