package api

import (
	"context"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
)

func saveRule(ctx context.Context, client *firestore.Client, w http.ResponseWriter, rule *Rule) {

	_, err := client.Collection(os.Getenv("DB_PREFIX")+"_rules").Doc(rule.Identifier).Set(ctx, &rule)
	if err != nil {
		log.Printf("Failed to update collection: %v", err)
		http.Error(w, "Failed to update collection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
