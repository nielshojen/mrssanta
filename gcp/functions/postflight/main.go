package postflight

import (
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

var validAPIKey = os.Getenv("API_KEY")

func init() {
	functions.HTTP("postflight", postflight)
}
