package f1livetiming

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/coder/websocket"
)

// TestNewClient ensures that the default configuration is pointing to the F1 Live Timing API and
// that the optional function parameters to set endpoints works correctly.
func TestNewClient(t *testing.T) {
	c := NewClient(WithLogger(testLogger(t)))

	expected := "https://livetiming.formula1.com"
	if c.httpBaseURL != expected {
		t.Errorf("Client.HTTPBaseURL was not defaulted to the correct value, expected '%s', found '%s'", expected, c.httpBaseURL)
	}
	expected = "wss://livetiming.formula1.com"
	if c.wsBaseURL != expected {
		t.Errorf("Client.WSBaseURL was not defaulted to the correct value, expected '%s', found '%s'", expected, c.wsBaseURL)
	}

	h := "http://test.com"
	w := httpToWS(t, "http://test.com")
	c = NewClient(WithHTTPBaseURL(h), WithWSBaseURL(w), WithLogger(testLogger(t)))
	if c.httpBaseURL != h {
		t.Errorf("Client.HTTPBaseURL was not set to the correct value, expected '%s', found '%s'", h, c.httpBaseURL)
	}
	if c.wsBaseURL != w {
		t.Errorf("Client.HTTPBaseURL was not set to the correct value, expected '%s', found '%s'", w, c.wsBaseURL)
	}
}

// TestNegotiate ensures that the connection token is correctly parsed from the F1 Live Timing
// `/negotiate` endpoint.
func TestNegotiate(t *testing.T) {
	ts := newWSTestServer(t)
	defer ts.Close()

	c := NewClient(WithHTTPBaseURL(ts.URL), WithLogger(testLogger(t)))
	c.Negotiate()

	e := "connection-token"
	if c.connectionToken != e {
		t.Errorf("Client.ConnectionToken expected '%s', found '%s'", e, c.connectionToken)
	}
}

// TestConnectWithoutNegotiate ensures that the client forces a Negotiate() function call before
// calling the Connect() function.
func TestConnectWithoutNegotiate(t *testing.T) {
	c := NewClient(WithLogger(testLogger(t)))

	err := c.Connect()

	if err == nil || !strings.Contains(err.Error(), "client.Negotiate()") {
		t.Errorf("Client.Connect() should require a successful Client.Negotiate")
	}
}

func TestConnectionSubscribe(t *testing.T) {
	i := make(chan struct{})
	ts := newWSTestServer(t, func() http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				t.Errorf("error setting up websocket in test")
			}

			_, msgBytes, err := conn.Read(context.Background())
			if err != nil {
				t.Errorf("error in test websocket server reading subscribe message - %s", err.Error())
			}

			msg := string(msgBytes)
			if !strings.Contains(msg, `"M": "Subscribe"`) {
				t.Errorf("expected subscribe message but found - %s", msg)
			}
			close(i)
		}
	}())
	defer ts.Close()

	c := NewClient(
		WithHTTPBaseURL(ts.URL),
		WithWSBaseURL(httpToWS(t, ts.URL)),
		WithLogger(testLogger(t)),
		WithInterruptChan(i),
	)

	err := c.Negotiate()
	if err != nil {
		t.Errorf("unexpected error negotiating connection")
	}
	err = c.Connect()
	if err != nil {
		t.Errorf("unexpected error negotiating connection")
	}
	<-c.Done
	fmt.Println("and we're done?")
}

/* Private Helper Functions
------------------------------------------------------------------------------------------------- */

// testLogger creates a new logger to be used in tests that writes all logs to /dev/null so they
// don't uglify the test output.
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// httpToWS is a helper function that takes an http(s) endpoint and converts it to a ws(s) endpoint.
func httpToWS(t *testing.T, u string) string {
	t.Helper()
	httpsRe := regexp.MustCompile("https://")
	httpRe := regexp.MustCompile("http://")

	wsUrl := httpsRe.ReplaceAllString(u, "wss://")
	return httpRe.ReplaceAllString(wsUrl, "ws://")
}

// newWSTestServer creates a mock server for testing that supports the negotiate and connect
// endpoints exposed by the F1 Live Timing API
func newWSTestServer(t *testing.T, wsHandlers ...http.HandlerFunc) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/signalr/negotiate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cookie", "test-cookie")
		fmt.Fprintln(w, `
    {
      "Url": "/signalr",
      "ConnectionToken": "connection-token",
      "ConnectionId": "connection-id",
      "KeepAliveTimeout": 20.0,
      "DisconnectTimeout": 30.0,
      "ConnectionTimeout": 110.0,
      "TryWebSockets": true,
      "ProtocolVersion": "1.5",
      "TransportConnectTimeout": 10.0,
      "LongPollDelay": 1.0
    }
    `)
	})

	mux.HandleFunc("/signalr/connect", wsHandlers[0])
	s := httptest.NewServer(mux)

	return s
}
