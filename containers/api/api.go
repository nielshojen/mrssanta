package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	apiKey := r.Header.Get("X-API-Key")
	if apiKey != validAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Unauthorized"}`)
		return
	}

	endpoint, id, ok := parseAPIPath(r.URL.Path)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	Infof("Request for endpoint=%q id=%q", endpoint, id)

	switch {
	case r.Method == http.MethodGet && endpoint == "" && id == "":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"message":"api container is running"}`)

	case r.Method == http.MethodGet && endpoint == "rules" && id == "":
		rules, err := getAllRules(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching rules: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(rules); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodGet && endpoint == "rules" && id != "":
		rules, err := getRuleByID(ctx, id)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching rules: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(rules); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodPost && endpoint == "rules" && id == "":
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			Warnf("Failed to read request body: %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		rules, err := createRules(ctx, w, reqBody)
		if err != nil {
			Warnf("Failed to create rules: %v", err)
			http.Error(w, fmt.Sprintf("Failed to create rules: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message": "Rules saved successfully", "count": %d}`, len(rules))

	case r.Method == http.MethodDelete && endpoint == "rules" && id != "":
		err := deleteRule(ctx, id)
		if errors.Is(err, errRuleNotFound) {
			http.Error(w, "Rule not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete rule: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message": "Rule deleted"}`)

	case r.Method == http.MethodPost && endpoint == "managedapp" && id != "":
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			Warnf("Failed to read request body: %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		err = assignRule(ctx, w, reqBody, id)
		if err != nil {
			Warnf("Failed to assign rule: %v", err)
			http.Error(w, fmt.Sprintf("Failed to create rules: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message": "Rule assigned successfully"}`)

	case r.Method == http.MethodDelete && endpoint == "managedapp" && id != "":
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			Warnf("Failed to read request body: %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		err = unassignRule(ctx, w, reqBody, id)
		if err != nil {
			Warnf("Failed to unassign rule: %v", err)
			http.Error(w, fmt.Sprintf("Failed to unassign rule: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message": "Rule unassigned successfully"}`)

	case r.Method == http.MethodGet && endpoint == "devices" && id == "":
		devices, err := getAllDevices(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching devices: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(devices); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodGet && endpoint == "devices" && id != "":
		device, err := getDeviceByID(ctx, id)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching device: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(device); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodPost && endpoint == "devices" && id == "":
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			Warnf("Failed to read request body: %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		devices, err := createDevice(ctx, w, reqBody)
		if err != nil {
			Warnf("Failed to create devices: %v", err)
			http.Error(w, fmt.Sprintf("Failed to create devices: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message": "Devices saved successfully", "count": %d}`, len(devices))

	case r.Method == http.MethodGet && endpoint == "events" && id == "":
		events, err := getAllEvents(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching events: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(events); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodGet && endpoint == "events" && id != "":
		events, err := getEventByID(ctx, id)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching events: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(events); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodDelete && endpoint == "bootstrap" && id != "":
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			Warnf("Failed to read request body: %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		err = unassignManagedAppRules(ctx, w, reqBody)
		if err != nil {
			Warnf("Failed to unassign rule: %v", err)
			http.Error(w, fmt.Sprintf("Failed to unassign rule: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message": "Rules unassigned successfully"}`)

	default:
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

func parseAPIPath(path string) (endpoint string, id string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "api" {
		return "", "", true
	}

	if len(parts) == 2 && parts[0] == "api" && parts[1] != "" {
		return parts[1], "", true
	}

	if len(parts) == 3 && parts[0] == "api" && parts[1] != "" && parts[2] != "" {
		return parts[1], parts[2], true
	}

	return "", "", false
}
