package main

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

  "net/http"

	"github.com/cert-manager/cert-manager/test/acme/dns"

  "github.com/jokajak/cert-manager-webhook-henet/utils/mocks"
)

var (
	zone = os.Getenv("TEST_ZONE_NAME")
	pollInterval = time.Second * 10
	propagationLimit = time.Minute * 6
)

func TestRunsSuite(t *testing.T) {
	// The manifest path should contain a file named config.json that is a
	// snippet of valid configuration that should be included on the
	// ChallengeRequest passed as part of the test cases.
	//

  // set up mock for HE API responses
  Client = &mocks.MockClient{}
  // build response JSON
  resp := `good`
  // create a new reader with that JSON
  r := io.NopCloser(bytes.NewReader([]byte(resp)))
  mocks.GetDoFunc = func(*http.Request) (*http.Response, error) {
    return &http.Response{
      StatusCode: 200,
      Body:       r,
    }, nil
  }

	// Uncomment the below fixture when implementing your custom DNS provider
	// Allow up-to 6 minutes for propagation, 5 minutes is the lowest TTL on HEnet
	// Since we have such a long propagation limit, we poll every 10 seconds
	fixture := dns.NewFixture(&HEnetDNSProviderSolver{},
		dns.SetResolvedZone(zone),
		dns.SetAllowAmbientCredentials(false),
		dns.SetManifestPath("testdata/henet"),
		dns.SetPollInterval(pollInterval),
		dns.SetPropagationLimit(propagationLimit),
	)
	// You must setup all the entries manually, run only the basic set
	fixture.RunBasic(t)

}
