package eventupload

import (
	"encoding/json"
	"fmt"
	"os"
)

// Log to GCP Cloud logging
func logEvent(event Event) error {

	logevent := event

	labels := Labels{
		Env:     os.Getenv("ENVIRONMENT"),
		App:     "mrssanta-eventupload",
		Service: "mrssanta-eventupload",
		Owner:   os.Getenv("OWNER"),
		Team:    os.Getenv("TEAM"),
		Version: os.Getenv("VERSION"),
	}

	logevent.Labels = &labels

	logData, err := json.Marshal(logevent)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	fmt.Println(string(logData))

	return err
}
