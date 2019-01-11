package contracts

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	// nop
}

// PactDir returns a path to the directory to store pact files (taken from PACT_DIR env variable)
func PactDir() string {
	return os.Getenv("PACT_DIR")
}

// PactFile returns a path to the generated pact file
func PactFile(pactConsumer string, pactProvider string) string {
	return fmt.Sprintf("%s/%s-%s.json", PactDir(), strings.ToLower(pactConsumer), strings.ToLower(pactProvider))
}
