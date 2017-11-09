package httpmonitor

import (
	"fmt"
	"net/http"

	"github.com/davecgh/go-spew/spew"
)

// TransportMonitor is a wrapper around the HTTP client's transport
// It collects usage stats of the request/responses, to be able to verify what how the HTTP client was actually used.
type TransportMonitor struct {
	transport http.RoundTripper
	Exchanges []Exchange
}

// Exchange a type to store the request's method and URL and the response's status code
type Exchange struct {
	RequestMethod string
	RequestURL    string
	StatusCode    int
	Error         error
}

// NewTransportMonitor returns a new Transport monitor
func NewTransportMonitor(t http.RoundTripper) *TransportMonitor {
	return &TransportMonitor{transport: t}
}

// ValidateExchanges verifies that the transport monitor contains the exact given sequence of exchanges
func (t *TransportMonitor) ValidateExchanges(exchanges ...Exchange) error {
	if len(t.Exchanges) != len(exchanges) {
		return fmt.Errorf("unexpected number of exchanges to compare: %d (actual) vs %d (expected)", len(t.Exchanges), len(exchanges))
	}
	for i, e := range exchanges {
		if t.Exchanges[i] != e {
			return fmt.Errorf("unexpected number of exchange at index #%d: %v vs %v", i, spew.Sdump(t.Exchanges[i]), spew.Sdump(e))
		}
	}
	return nil
}

// ValidateNoExchanges verifies that the transport monitor does not contain any record of exchanges
func (t *TransportMonitor) ValidateNoExchanges() error {
	if len(t.Exchanges) != 0 {
		return fmt.Errorf("unexpected number of exchanges: %d (actual)", len(t.Exchanges))
	}
	return nil
}

// RoundTrip implements the http.RoundTripper#RoundTrip(*http.Request) (*http.Response, error) function,
// It delegates the call to the underlying RoundTripper of this monitor, and keps track of the request/response
// exhanges data.
func (t *TransportMonitor) RoundTrip(request *http.Request) (*http.Response, error) {
	e := Exchange{
		RequestMethod: request.Method,
		RequestURL:    request.URL.String(),
	}
	response, err := t.transport.RoundTrip(request)
	// collect stats about the request and the response
	if err != nil {
		e.Error = err
	} else {
		e.StatusCode = response.StatusCode
	}
	t.Exchanges = append(t.Exchanges, e)
	return response, err
}
