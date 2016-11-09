package env

import "os"

// Environment is the type of the possible environment values.
type Environment string

const (
	// Production is the production environment.
	Production Environment = "production"
	// QA is the qa environment.
	QA Environment = "qa"
)

// Current is the current environment set after init.
var Current Environment

func init() {
	if os.Getenv("ENVIRONMENT") == "production" {
		Current = Environment("production")
	} else {
		Current = Environment("qa")
	}
}
