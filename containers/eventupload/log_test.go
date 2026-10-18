package eventupload

import (
	"strings"
	"testing"
)

func TestApplyLogstashFieldsUsesDefaults(t *testing.T) {
	t.Setenv("DD_SOURCE", "")
	t.Setenv("DD_SERVICE", "")
	t.Setenv("SERVICE_NAME", "")
	t.Setenv("K_SERVICE", "")
	t.Setenv("ENVIRONMENT", "")
	t.Setenv("ENV", "")
	t.Setenv("DD_ENV", "")

	event := Event{}
	applyLogstashFields(&event)

	if event.Service != defaultLogstashSource {
		t.Fatalf("Service = %q, want %q", event.Service, defaultLogstashSource)
	}
	if event.DDSource != defaultLogstashSource {
		t.Fatalf("DDSource = %q, want %q", event.DDSource, defaultLogstashSource)
	}

	assertTag(t, event.DDTags, "env:"+defaultEnvironment)
	assertTag(t, event.DDTags, "app:"+defaultLogstashSource)
	assertTag(t, event.DDTags, "service:"+defaultLogstashService)
	assertTag(t, event.DDTags, "source:"+defaultLogstashSource)
}

func TestApplyLogstashFieldsUsesEnvOverrides(t *testing.T) {
	t.Setenv("DD_SOURCE", "custom-source")
	t.Setenv("DD_SERVICE", "custom-service")
	t.Setenv("SERVICE_NAME", "")
	t.Setenv("K_SERVICE", "")
	t.Setenv("ENVIRONMENT", "prod")
	t.Setenv("OWNER", "owner,with,commas")
	t.Setenv("TEAM", "mac")
	t.Setenv("VERSION", "1.2.3")
	t.Setenv("COSTCENTER", "123")

	event := Event{}
	applyLogstashFields(&event)

	if event.Service != "custom-source" {
		t.Fatalf("Service = %q, want %q", event.Service, "custom-source")
	}
	if event.DDSource != "custom-source" {
		t.Fatalf("DDSource = %q, want %q", event.DDSource, "custom-source")
	}

	assertTag(t, event.DDTags, "env:prod")
	assertTag(t, event.DDTags, "app:custom-source")
	assertTag(t, event.DDTags, "service:custom-service")
	assertTag(t, event.DDTags, "source:custom-source")
	assertTag(t, event.DDTags, "owner:owner_with_commas")
	assertTag(t, event.DDTags, "team:mac")
	assertTag(t, event.DDTags, "costcenter:123")
	assertTag(t, event.DDTags, "version:1.2.3")
}

func assertTag(t *testing.T, tags, want string) {
	t.Helper()

	for _, tag := range strings.Split(tags, ",") {
		if tag == want {
			return
		}
	}

	t.Fatalf("tags %q missing %q", tags, want)
}
