package postflight

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// maxSyncBodyBytes caps the size of a sync request body, before and after
// decompression.
const maxSyncBodyBytes = 32 << 20

// wireFormat is the encoding Santa used for a request. Santa expects the
// response in the same encoding.
type wireFormat int

const (
	wireJSON wireFormat = iota
	wireProto
)

const protoContentType = "application/x-protobuf"

var errUnsupportedContentType = errors.New("unsupported Content-Type")

// readSyncRequest decodes a Santa sync request into msg. It undoes the
// Content-Encoding (none, gzip or deflate) and unmarshals either JSON or binary
// protobuf depending on the Content-Type (SyncEnableProtoTransfer). Unknown
// JSON fields are ignored so newer agents keep working.
func readSyncRequest(r *http.Request, msg proto.Message) (wireFormat, error) {
	format, err := requestWireFormat(r.Header.Get("Content-Type"))
	if err != nil {
		return format, err
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxSyncBodyBytes+1))
	if err != nil {
		return format, fmt.Errorf("read body: %w", err)
	}
	if len(body) > maxSyncBodyBytes {
		return format, errors.New("request body too large")
	}

	data, err := decodeContent(body, r.Header.Get("Content-Encoding"))
	if err != nil {
		return format, err
	}

	if format == wireProto {
		if err := proto.Unmarshal(data, msg); err != nil {
			return format, fmt.Errorf("decode protobuf: %w", err)
		}
		return format, nil
	}

	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, msg); err != nil {
		return format, fmt.Errorf("decode JSON: %w", err)
	}
	return format, nil
}

// writeSyncResponse encodes msg in the format the request used.
func writeSyncResponse(w http.ResponseWriter, format wireFormat, msg proto.Message) error {
	var (
		data        []byte
		err         error
		contentType string
	)

	if format == wireProto {
		data, err = proto.Marshal(msg)
		contentType = protoContentType
	} else {
		data, err = protojson.Marshal(msg)
		contentType = "application/json"
	}
	if err != nil {
		http.Error(w, `{"error": "Failed to encode response"}`, http.StatusInternalServerError)
		return fmt.Errorf("encode response: %w", err)
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	return err
}

func requestWireFormat(contentType string) (wireFormat, error) {
	if strings.TrimSpace(contentType) == "" {
		return wireJSON, nil
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return wireJSON, fmt.Errorf("%w: %q", errUnsupportedContentType, contentType)
	}

	switch mediaType {
	case "application/json":
		return wireJSON, nil
	case protoContentType, "application/protobuf":
		return wireProto, nil
	default:
		return wireJSON, fmt.Errorf("%w: %q", errUnsupportedContentType, mediaType)
	}
}

// decodeContent undoes the Content-Encoding. Santa labels zlib-wrapped data as
// "deflate", but raw deflate is accepted too. Without a Content-Encoding header
// a zlib body is still detected, which keeps older agents working.
func decodeContent(body []byte, contentEncoding string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(contentEncoding)) {
	case "", "identity":
		if looksLikeZlib(body) {
			if data, err := readAllLimited(zlib.NewReader(bytes.NewReader(body))); err == nil {
				return data, nil
			}
		}
		return body, nil
	case "gzip", "x-gzip":
		return readAllLimited(gzip.NewReader(bytes.NewReader(body)))
	case "deflate":
		if data, err := readAllLimited(zlib.NewReader(bytes.NewReader(body))); err == nil {
			return data, nil
		}
		return readAllLimited(flate.NewReader(bytes.NewReader(body)), nil)
	default:
		return nil, fmt.Errorf("unsupported Content-Encoding %q", contentEncoding)
	}
}

// looksLikeZlib reports whether data starts with a valid zlib header.
func looksLikeZlib(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	return data[0]&0x0f == 8 && (uint16(data[0])<<8|uint16(data[1]))%31 == 0
}

func readAllLimited(reader io.Reader, err error) ([]byte, error) {
	if err != nil {
		return nil, fmt.Errorf("decompress body: %w", err)
	}
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}

	data, err := io.ReadAll(io.LimitReader(reader, maxSyncBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("decompress body: %w", err)
	}
	if len(data) > maxSyncBodyBytes {
		return nil, errors.New("decompressed body too large")
	}
	return data, nil
}
