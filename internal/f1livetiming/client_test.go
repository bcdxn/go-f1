package f1livetiming

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
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
	ts := newWSTestServer(t, defaultWSHandler(t))
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
	// Connect to the test websocket server
	go c.Connect()
	// Wait for the client to close the connection
	err := <-c.DoneCh()
	if err == nil || !strings.Contains(err.Error(), "client.Negotiate()") {
		t.Errorf("Client.Connect() should require a successful Client.Negotiate")
	}
}

// TestConnectionSubscribe tests that the client sends the proper 'subscribe' message to the server
// to kickoff the live-timing websocket communication.
func TestConnectionSubscribe(t *testing.T) {
	ts := newWSTestServer(t, func() http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				t.Errorf("error setting up websocket in test")
			}

			running := true
			for running {
				_, msgBytes, err := conn.Read(context.Background())
				if err != nil && websocket.CloseStatus(err) != -1 {
					// received a close connection msg from the client completing the close handhake
					running = false
				} else if err != nil {
					running = false
					t.Errorf("error in test websocket server reading subscribe message - %s", err.Error())
				} else {
					msg := string(msgBytes)
					if !strings.Contains(msg, `\"M\": \"Subscribe\"`) {
						t.Errorf("expected subscribe message but found - %s", msg)
					}
					// send back a close message to start the close handshake
					conn.Close(websocket.StatusNormalClosure, "server closed connection")
					running = false
				}
			}
		}
	}())
	defer ts.Close()

	c := NewClient(
		WithHTTPBaseURL(ts.URL),
		WithWSBaseURL(httpToWS(t, ts.URL)),
		WithLogger(testLogger(t)),
	)

	err := c.Negotiate()
	if err != nil {
		t.Errorf("unexpected error negotiating connection")
	}

	go c.Connect()
	err = <-c.DoneCh()
	if err != nil {
		t.Errorf("unexpected error negotiating connection")
	}
}

// TestDriverListMsg tests that the client handles the DriverList message correct and writes its own
// domain-oriented data to the driver channel.
func TestDriverListMsg(t *testing.T) {
	ts := newWSTestServer(t, defaultWSHandler(t, "DriverList"))
	c := NewClient(
		WithHTTPBaseURL(ts.URL),
		WithWSBaseURL(httpToWS(t, ts.URL)),
		WithLogger(testLogger(t)),
	)

	driverList := driverList(t)
	c.Negotiate()
	go c.Connect()
	max := len(driverList)
	count := 0
	for {
		d := <-c.DriverCh()
		count++
		if d.Name != driverList[strconv.Itoa(d.Number)].FullName {
			t.Errorf("expected name %s but found %s", driverList["4"].FullName, d.Name)
		}
		number, err := strconv.Atoi(driverList[strconv.Itoa(d.Number)].RacingNumber)
		if err != nil {
			t.Errorf("unexpected racing number format in testdata - %s", driverList[strconv.Itoa(d.Number)].RacingNumber)
		}
		if d.Number != number {
			t.Errorf("expected name %d but found %d", number, d.Number)
		}
		if count >= max {
			break
		}
	}
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
func newWSTestServer(t *testing.T, wsHandler http.HandlerFunc) *httptest.Server {
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

	mux.HandleFunc("/signalr/connect", wsHandler)
	s := httptest.NewServer(mux)

	return s
}

// defaultWSeHandler returns a default handler for the test websocket server. Given a list of
// message types, it will read the test messages of that type and write them to the websocket afte
// it receives the 'subscribe' from the client.
func defaultWSHandler(t *testing.T, messageTypes ...string) http.HandlerFunc {
	dir := testdataDir()
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Errorf("error finding files in 'testdata'")
	}

	msgs := make([][]byte, 0)
	for _, messageType := range messageTypes {
		// filter the messages that match the given messageType
		msgRe := regexp.MustCompile("msg-" + strings.ToLower(messageType) + `(-[\d]+)?.json`)
		for _, file := range files {
			if msgRe.MatchString(file.Name()) {
				f := path.Join(dir, file.Name())
				msg, err := os.ReadFile(f)
				if err != nil {
					t.Errorf("error reading file %s", f)
				}
				msgs = append(msgs, msg)
			}
		}

	}

	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("error setting up websocket in test")
		}

		_, _, err = conn.Read(context.Background())
		if err != nil {
			t.Errorf("error in test websocket server reading subscribe message - %s", err.Error())
		}

		// once we have the subscribe message, we can write all of the messages to to the socket
		for _, msg := range msgs {
			err = conn.Write(context.Background(), websocket.MessageText, msg)
			if err != nil {
				t.Error("error writing test message to websocket", err)
			}
		}
	}
}

// getTestdataDir gets the testdata directory path relative to the invocation of the tests.
func testdataDir() string {
	_, p, _, _ := runtime.Caller(0)
	return path.Join(filepath.Dir(p), "testdata")
}

// driverList parses the driverlist test data file for matching in tests
func driverList(t *testing.T) map[string]driverData {
	td := testdataDir()
	contents, err := os.ReadFile(path.Join(td, "msg-driverlist-0.json"))
	if err != nil {
		t.Errorf("error reading test msg file")
	}
	var changeMessage struct {
		M []struct {
			A []any
		}
	}
	err = json.Unmarshal(contents, &changeMessage)
	if err != nil {
		t.Errorf("test data is in invalid format; expected change data message but found - %s", string(contents))
	}

	s, err := json.Marshal(changeMessage.M[0].A[1])
	if err != nil {
		t.Errorf("test data is in invalid format; unable to marshal message %s", string(contents))
	}
	var d map[string]driverData
	err = json.Unmarshal(s, &d)
	if err != nil {
		t.Errorf("test data is in invalid format; expected map of driver data but found %s", string(s))
	}

	return d
}
