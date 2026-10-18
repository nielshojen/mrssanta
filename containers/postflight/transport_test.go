package postflight

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"net/http"
	"net/http/httptest"
	"testing"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func zlibBytes(t *testing.T, data []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, err := w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func gzipBytes(t *testing.T, data []byte) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func rawDeflateBytes(t *testing.T, data []byte) []byte {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	require.NoError(t, err)
	_, err = w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func syncRequest(body []byte, contentType, contentEncoding string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/santa/preflight/machine-1", bytes.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	if contentEncoding != "" {
		r.Header.Set("Content-Encoding", contentEncoding)
	}
	return r
}

// santaJSON mirrors what Santa's protobuf JSON printer sends: explicit
// json_name values where the proto sets one, lowerCamelCase otherwise.
const santaJSON = `{"serial_num":"C02ABC","hostname":"mac.example.com","client_mode":"LOCKDOWN",` +
	`"request_clean_sync":true,"sipStatus":127,"rulesHash":"abc123","some_future_field":1}`

func TestReadSyncRequestEncodings(t *testing.T) {
	plain := []byte(santaJSON)

	tests := []struct {
		name            string
		body            []byte
		contentEncoding string
	}{
		{name: "uncompressed", body: plain},
		{name: "deflate (zlib wrapped)", body: zlibBytes(t, plain), contentEncoding: "deflate"},
		{name: "deflate (raw)", body: rawDeflateBytes(t, plain), contentEncoding: "deflate"},
		{name: "gzip", body: gzipBytes(t, plain), contentEncoding: "gzip"},
		{name: "zlib without header", body: zlibBytes(t, plain)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req syncv1.PreflightRequest
			format, err := readSyncRequest(syncRequest(tt.body, "application/json", tt.contentEncoding), &req)

			require.NoError(t, err)
			assert.Equal(t, wireJSON, format)
			assert.Equal(t, "C02ABC", req.GetSerialNumber())
			assert.Equal(t, syncv1.ClientMode_LOCKDOWN, req.GetClientMode())
			assert.True(t, req.GetRequestCleanSync())
			assert.Equal(t, uint32(127), req.GetSipStatus(), "lowerCamelCase field must be decoded")
			assert.Equal(t, "abc123", req.GetRulesHash())
		})
	}
}

func TestReadSyncRequestProto(t *testing.T) {
	data, err := proto.Marshal(syncv1.PreflightRequest_builder{
		SerialNumber: "C02ABC",
		ClientMode:   syncv1.ClientMode_MONITOR,
	}.Build())
	require.NoError(t, err)

	var req syncv1.PreflightRequest
	format, err := readSyncRequest(syncRequest(zlibBytes(t, data), "application/x-protobuf", "deflate"), &req)

	require.NoError(t, err)
	assert.Equal(t, wireProto, format)
	assert.Equal(t, "C02ABC", req.GetSerialNumber())
	assert.Equal(t, syncv1.ClientMode_MONITOR, req.GetClientMode())
}

func TestReadSyncRequestRejectsBadInput(t *testing.T) {
	tests := []struct {
		name            string
		body            []byte
		contentType     string
		contentEncoding string
	}{
		{name: "unsupported content type", body: []byte(santaJSON), contentType: "text/plain"},
		{name: "unsupported content encoding", body: []byte(santaJSON), contentType: "application/json", contentEncoding: "br"},
		{name: "corrupt gzip", body: []byte("not gzip"), contentType: "application/json", contentEncoding: "gzip"},
		{name: "invalid JSON", body: []byte("{"), contentType: "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req syncv1.PreflightRequest
			_, err := readSyncRequest(syncRequest(tt.body, tt.contentType, tt.contentEncoding), &req)
			assert.Error(t, err)
		})
	}
}

func TestReadSyncRequestDefaultsToJSON(t *testing.T) {
	var req syncv1.PreflightRequest
	format, err := readSyncRequest(syncRequest([]byte(santaJSON), "", ""), &req)

	require.NoError(t, err)
	assert.Equal(t, wireJSON, format)
	assert.Equal(t, "C02ABC", req.GetSerialNumber())
}

func TestWriteSyncResponse(t *testing.T) {
	resp := syncv1.PreflightResponse_builder{
		ClientMode:              syncv1.ClientMode_LOCKDOWN,
		SyncType:                syncv1.SyncType_CLEAN_ALL.Enum(),
		FullSyncIntervalSeconds: 600,
	}.Build()

	t.Run("json", func(t *testing.T) {
		rec := httptest.NewRecorder()
		require.NoError(t, writeSyncResponse(rec, wireJSON, resp))

		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Body.String(), `"LOCKDOWN"`)
		assert.Contains(t, rec.Body.String(), `"CLEAN_ALL"`)
		assert.Contains(t, rec.Body.String(), `"full_sync_interval"`)
	})

	t.Run("proto", func(t *testing.T) {
		rec := httptest.NewRecorder()
		require.NoError(t, writeSyncResponse(rec, wireProto, resp))

		assert.Equal(t, protoContentType, rec.Header().Get("Content-Type"))
		var decoded syncv1.PreflightResponse
		require.NoError(t, proto.Unmarshal(rec.Body.Bytes(), &decoded))
		assert.True(t, proto.Equal(resp, &decoded))
	})
}
