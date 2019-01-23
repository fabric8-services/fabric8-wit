package main

import (
	"os"
	"regexp"

	"github.com/fabric8-services/fabric8-wit/test/contracts"
)

func main() {
	pactFiles := os.Args[1]
	pactBrokerURL := os.Getenv("PACT_BROKER_URL")
	pactBrokerUsername := os.Getenv("PACT_BROKER_USERNAME")
	pactBrokerPassword := os.Getenv("PACT_BROKER_PASSWORD")
	pactVersion := os.Args[2]
	tags := os.Args[3]

	re := regexp.MustCompile("[;\n]")

	contracts.PublishPactFileToBroker(
		re.Split(pactFiles, -1),
		pactBrokerURL,
		pactBrokerUsername,
		pactBrokerPassword,
		pactVersion,
		re.Split(tags, -1),
	)
}
