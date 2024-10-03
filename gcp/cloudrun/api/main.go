package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
)

var client *firestore.Client
var firestoreDatabasePath string

func main() {
	var err error
	ctx := context.Background()

	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		log.Fatal("GCP_PROJECT environment variable is not set")
	}

	firestoreDatabasePath = os.Getenv("FIRESTORE_DATABASE")
	if firestoreDatabasePath == "" {
		log.Fatal("FIRESTORE_DATABASE environment variable is not set")
	}

	dbPrefix := os.Getenv("DB_PREFIX")
	if dbPrefix == "" {
		log.Fatal("DB_PREFIX environment variable is not set")
	}

	client, err = firestore.NewClientWithDatabase(ctx, projectID, firestoreDatabasePath)
	if err != nil {
		log.Fatalf("Error initializing Cloud Firestore client: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := mux.NewRouter()
	r.HandleFunc("/", rootHandler)
	r.HandleFunc("/rules", rulesHandler)
	r.HandleFunc("/rules/{id}", ruleHandler)
	r.HandleFunc("/binaries", binariesHandler)

	log.Println("Mrs Santa REST API listening on port", port)
	cors := handlers.CORS(
		handlers.AllowedHeaders([]string{"X-Requested-With", "Authorization", "Origin"}),
		handlers.AllowedOrigins([]string{"https://storage.googleapis.com"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "OPTIONS", "PATCH", "CONNECT"}),
	)

	if err := http.ListenAndServe(":"+port, cors(r)); err != nil {
		log.Fatalf("Error launching Mrs Santa REST API server: %v", err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "{status: 'running'}")
}

func rulesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	w.Header().Set("Content-Type", "application/json")
	rule, err := getRules(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"status": "fail", "data": '%s'}`, err)
		return
	}
	if rule == nil {
		w.WriteHeader(http.StatusNotFound)
		msg := fmt.Sprintf("`Rules not found`")
		fmt.Fprintf(w, `{"status": "fail", "data": {"title": %s}}`, msg)
		return
	}
	data, err := json.Marshal(rule)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"status": "fail", "data": "Unable to fetch rules: %s"}`, err)
		return
	}
	fmt.Fprintf(w, `{"status": "success", "data": %s}`, data)
}

func ruleHandler(w http.ResponseWriter, r *http.Request) {
	identifier := mux.Vars(r)["id"]
	ctx := context.Background()
	w.Header().Set("Content-Type", "application/json")
	rule, err := getRule(ctx, identifier)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"status": "fail", "data": '%s'}`, err)
		return
	}
	if rule == nil {
		w.WriteHeader(http.StatusNotFound)
		msg := fmt.Sprintf("`Rule \"%s\" not found`", identifier)
		fmt.Fprintf(w, `{"status": "fail", "data": {"title": %s}}`, msg)
		return
	}
	data, err := json.Marshal(rule)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"status": "fail", "data": "Unable to fetch rule: %s"}`, err)
		return
	}
	fmt.Fprintf(w, `{"status": "success", "data": %s}`, data)
}

func binariesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	w.Header().Set("Content-Type", "application/json")
	binaries, err := getBinaries(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"status": "fail", "data": '%s'}`, err)
		return
	}
	if binaries == nil {
		w.WriteHeader(http.StatusNotFound)
		msg := fmt.Sprintf("`Binaries not found`")
		fmt.Fprintf(w, `{"status": "fail", "data": {"title": %s}}`, msg)
		return
	}
	data, err := json.Marshal(binaries)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"status": "fail", "data": "Unable to fetch rules: %s"}`, err)
		return
	}
	fmt.Fprintf(w, `{"status": "success", "data": %s}`, data)
}

type Rule struct {
	CreationTime          string `firestore:"creation_time" json:"creation_time"`
	CustomMessage         string `firestore:"custom_msg" json:"custom_msg,omitempty"`
	CustomURL             string `firestore:"custom_url" json:"custom_url,omitempty"`
	FileBundleBinaryCount string `firestore:"file_bundle_binary_count" json:"file_bundle_binary_count,omitempty"`
	FileBundleHash        string `firestore:"file_bundle_hash" json:"file_bundle_hash,omitempty"`
	Identifier            string `firestore:"identifier" json:"identifier"`
	Policy                string `firestore:"policy" json:"policy"`
	RuleType              string `firestore:"rule_type" json:"rule_type"`
	Scope                 string `firestore:"scope" json:"scope"`
}

func getRules(ctx context.Context) ([]*Rule, error) {
	query := client.Collection(os.Getenv("DB_PREFIX") + "_rules")
	iter := query.Documents(ctx)

	var rules []*Rule
	for {
		var r Rule
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		err = doc.DataTo(&r)
		if err != nil {
			return nil, err
		}
		rules = append(rules, &r)
	}
	return rules, nil
}

func getRule(ctx context.Context, identifier string) (*Rule, error) {
	query := client.Collection(os.Getenv("DB_PREFIX")+"_rules").Where("identifier", "==", identifier)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	if len(docs) == 0 {
		return nil, nil
	}

	var rule Rule
	if err := docs[0].DataTo(&rule); err != nil {
		return nil, err
	}

	return &rule, nil
}

func getBinaries(ctx context.Context) (data []map[string]interface{}, err error) {
	query := client.Collection(os.Getenv("DB_PREFIX") + "_binaries")
	iter := query.Documents(ctx)

	var binaries []map[string]interface{}
	for {
		var b []map[string]interface{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		err = doc.DataTo(&b)
		if err != nil {
			return nil, err
		}
		binaries = append(binaries, b...)
	}
	return binaries, nil
}
