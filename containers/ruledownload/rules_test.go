package ruledownload

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestFullSyncReason(t *testing.T) {
	lastSync := primitive.NewDateTimeFromTime(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))

	tests := []struct {
		name   string
		device Device
		want   bool
		reason string
	}{
		{
			name:   "pending clean all sends full rule set",
			device: Device{LastSync: lastSync, PendingSyncType: "CLEAN_ALL"},
			want:   true,
			reason: "pending_sync_type=CLEAN_ALL",
		},
		{
			name:   "pending clean sends full rule set",
			device: Device{LastSync: lastSync, PendingSyncType: "CLEAN"},
			want:   true,
			reason: "pending_sync_type=CLEAN",
		},
		{
			name:   "needs_clean_sync sends full rule set",
			device: Device{LastSync: lastSync, NeedsCleanSync: true},
			want:   true,
			reason: "needs_clean_sync",
		},
		{
			name:   "request_clean_sync sends full rule set",
			device: Device{LastSync: lastSync, RequestCleanSync: true},
			want:   true,
			reason: "request_clean_sync",
		},
		{
			name:   "first sync sends full rule set",
			device: Device{PendingSyncType: "NORMAL"},
			want:   true,
			reason: "first_sync",
		},
		{
			name:   "pending normal sync is incremental",
			device: Device{LastSync: lastSync, PendingSyncType: "NORMAL"},
			want:   false,
			reason: "incremental",
		},
		{
			name:   "no pending sync type is incremental",
			device: Device{LastSync: lastSync},
			want:   false,
			reason: "incremental",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := fullSyncReason(&tt.device)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.reason, reason)
		})
	}
}
