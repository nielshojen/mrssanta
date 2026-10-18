package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Shapes taken from Santa's metric format test data
// (Source/santametricservice/Formats/testdata/json).
const rawJSONMetrics = `{
	"metrics": {
		"/santa/rules": {
			"type": 7,
			"description": "Number of rules",
			"fields": {"rule_type": [{"value": "binary", "data": 1}]}
		}
	},
	"root_labels": {"hostname": "testHost", "username": "testUser"}
}`

const monarchJSONMetrics = `{
	"metricsCollection": [{
		"rootLabels": [{"key": "hostname", "stringValue": "testHost"}],
		"metricsDataSet": [{"metricName": "/santa/rules", "valueType": "INT64", "data": [{"int64Value": 1}]}]
	}]
}`

func postMetrics(method, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "/metrics", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	metricsHandler(w, r)
	return w
}

func TestParseMetricsRecognisesSantaFormats(t *testing.T) {
	format, doc := parseMetrics([]byte(rawJSONMetrics))
	assert.Equal(t, "rawjson", format)
	assert.Contains(t, doc, "root_labels")

	format, doc = parseMetrics([]byte(monarchJSONMetrics))
	assert.Equal(t, "monarchjson", format)
	assert.Contains(t, doc, "metricsCollection")
}

func TestParseMetricsRejectsOtherBodies(t *testing.T) {
	for name, body := range map[string]string{
		"not JSON":                 "hello",
		"JSON array":               `[1, 2, 3]`,
		"unrelated object":         `{"message": "spam"}`,
		"metrics is not an object": `{"metrics": "spam"}`,
		"collection is not a list": `{"metricsCollection": {"a": 1}}`,
		"empty":                    "",
	} {
		t.Run(name, func(t *testing.T) {
			format, doc := parseMetrics([]byte(body))
			assert.Empty(t, format)
			assert.Nil(t, doc)
		})
	}
}

func TestMetricsHandlerAcceptsSantaMetrics(t *testing.T) {
	for _, body := range []string{rawJSONMetrics, monarchJSONMetrics} {
		w := postMetrics(http.MethodPost, body)
		assert.Equal(t, http.StatusOK, w.Code, "Santa treats anything but 200 as a failed export")
	}

	assert.Equal(t, http.StatusOK, postMetrics(http.MethodPut, rawJSONMetrics).Code)
}

func TestMetricsHandlerRejectsJunk(t *testing.T) {
	assert.Equal(t, http.StatusBadRequest, postMetrics(http.MethodPost, `{"message": "spam"}`).Code)
	assert.Equal(t, http.StatusBadRequest, postMetrics(http.MethodPost, "not json").Code)
}

func TestMetricsHandlerRejectsOversizedBodies(t *testing.T) {
	body := `{"metrics": {"x": "` + strings.Repeat("a", maxMetricsBodyBytes) + `"}}`

	assert.Equal(t, http.StatusRequestEntityTooLarge, postMetrics(http.MethodPost, body).Code)
}

func TestMetricsHandlerRejectsOtherMethods(t *testing.T) {
	assert.Equal(t, http.StatusMethodNotAllowed, postMetrics(http.MethodGet, "").Code)
}
