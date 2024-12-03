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
	"sync"
	"testing"

	"github.com/bcdxn/f1cli/internal/domain"
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

// TestDriverListMsg tests that the client handles the DriverList message correctly by updating
// the client's internal state store and writing a notification to the driver channel.
func TestDriverListMsg(t *testing.T) {
	ts := newWSTestServer(t, defaultWSHandler(t, "DriverList"))
	c := NewClient(
		WithHTTPBaseURL(ts.URL),
		WithWSBaseURL(httpToWS(t, ts.URL)),
		WithLogger(testLogger(t)),
	)

	expectedDriverList := driverList(t)
	c.Negotiate()
	go c.Connect()

	<-c.DriverCh()
	actualDriverList := c.DriversState()

	for _, n := range []uint8{11, 44, 55, 77} {
		ns := strconv.Itoa(int(n))

		if actualDriverList[n].Number != n {
			t.Errorf("expected number %d but found %d", n, actualDriverList[n].Number)
		}
		if actualDriverList[n].Name != *expectedDriverList[ns].FullName {
			t.Errorf("expected name %s but found %s", *expectedDriverList[ns].FullName, actualDriverList[n].Name)
		}
		if actualDriverList[n].ShortName != *expectedDriverList[ns].ShortName {
			t.Errorf("expected short name %s but found %s", *expectedDriverList[ns].ShortName, actualDriverList[n].ShortName)
		}
		if actualDriverList[n].TeamName != *expectedDriverList[ns].TeamName {
			t.Errorf("expected team name %s but found %s", *expectedDriverList[ns].TeamName, actualDriverList[n].TeamName)
		}
		if actualDriverList[n].TeamColor != *expectedDriverList[ns].TeamColour {
			t.Errorf("expected team color %s but found %s", *expectedDriverList[ns].TeamColour, actualDriverList[n].TeamColor)
		}
	}
	c.Close()
}

// TestTimingDataMsg tests that the client handles the TimingData message correctly by updating
// the client's internal state store and writing a notification to the driver channel.
func TestTimingDataMsg(t *testing.T) {
	ts := newWSTestServer(t, defaultWSHandler(t, "TimingData"))
	c := NewClient(
		WithHTTPBaseURL(ts.URL),
		WithWSBaseURL(httpToWS(t, ts.URL)),
		WithLogger(testLogger(t)),
	)

	c.Negotiate()
	go c.Connect()

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-c.DriverCh()
	}()
	go func() {
		defer wg.Done()
		<-c.EventCh()
	}()
	wg.Wait()
	actualDriverList := c.DriversState()
	weekendEvent := c.EventState()
	if actualDriverList[55].Number != 55 {
		t.Errorf("expected driver number 55 but found %d", actualDriverList[55].Number)
	}
	if actualDriverList[55].LeaderGap != "+12.562" {
		t.Errorf("expected LeaderGap of '+12.562' but found '%s'", actualDriverList[55].LeaderGap)
	}
	if actualDriverList[55].IntervalGap != "+3.421" {
		t.Errorf("expected IntervalGap of '+3.421' but found '%s'", actualDriverList[55].IntervalGap)
	}
	if weekendEvent.Session.FastestLapOwner != 55 {
		t.Errorf("unexpected fastest lap owner - %d", weekendEvent.Session.FastestLapOwner)
	}
	c.Close()
}

// TestSessionInfoMsg tests that the client handles the SessionInfo message correctly by updating
// the client's internal state store and writing a notification to the event channel.
func TestSessionInfoMsg(t *testing.T) {
	ts := newWSTestServer(t, defaultWSHandler(t, "SessionInfo"))
	c := NewClient(
		WithHTTPBaseURL(ts.URL),
		WithWSBaseURL(httpToWS(t, ts.URL)),
		WithLogger(testLogger(t)),
	)
	c.Negotiate()
	go c.Connect()
	// wait for notification on Race Weekend Event channel
	<-c.EventCh()

	event := c.EventState()
	if event.FullName != "FORMULA 1 PIRELLI UNITED STATES GRAND PRIX 2024" {
		t.Errorf("unexpected race weekend event name - %s", event.FullName)
	}
	if event.RoundNumber != 19 {
		t.Errorf("unexpected race weekend event round # - %d", event.RoundNumber)
	}
	if event.CountryCode != "USA" {
		t.Errorf("unexpected race weekend event country code - %s", event.CountryCode)
	}
	if event.Session.Name != "Race" {
		t.Errorf("unexpected session name - %s", event.Session.Name)
	}
	if event.Session.StartDate.UTC().Format("2006-01-02T15:04:05") != "2024-10-20T19:00:00" {
		t.Errorf("unexpected session start date - %s", event.Session.StartDate.UTC().Format("2006-01-02T15:04:05"))
	}
	if event.Session.EndDate.UTC().Format("2006-01-02T15:04:05") != "2024-10-20T21:00:00" {
		t.Errorf("unexpected session start date - %s", event.Session.EndDate.UTC().Format("2006-01-02T15:04:05"))
	}

	c.Close()
}

// TestLapCountMsg tests that the client handles LapCount messages from the F1 LiveTiming API
// correctly by updating the client's internal state store and writing a notification to the event
// channel.
func TestLapCountMsg(t *testing.T) {
	ts := newWSTestServer(t, defaultWSHandler(t, "SessionInfo", "LapCount"))
	c := NewClient(
		WithHTTPBaseURL(ts.URL),
		WithWSBaseURL(httpToWS(t, ts.URL)),
		WithLogger(testLogger(t)),
	)
	c.Negotiate()
	go c.Connect()
	// wait for three notifications on the event channel
	<-c.EventCh()
	<-c.EventCh()
	event := c.EventState()
	if event.Session.CurrentLap != 1 {
		t.Errorf("unexpected lap count - %d", event.Session.CurrentLap)
	}
	if event.Session.TotalLaps != 56 {
		t.Errorf("unexpected lap count - %d", event.Session.TotalLaps)
	}
	<-c.EventCh()
	event = c.EventState()
	if event.Session.CurrentLap != 2 {
		t.Errorf("unexpected lap count - %d", event.Session.CurrentLap)
	}
	if event.Session.TotalLaps != 56 {
		t.Errorf("unexpected lap count - %d", event.Session.TotalLaps)
	}
	c.Close()
}

// TestTimingAppDataMsg tests that the client handles TestTimingAppData messages from the F1
// LiveTiming API correctly by updating the client's internal state store and writing a notification
// to the driver channel.
func TestTimingAppDataMsg(t *testing.T) {
	ts := newWSTestServer(t, defaultWSHandler(t, "TimingAppData"))
	c := NewClient(
		WithHTTPBaseURL(ts.URL),
		WithWSBaseURL(httpToWS(t, ts.URL)),
		WithLogger(testLogger(t)),
	)
	c.Negotiate()
	go c.Connect()
	// wait for the driver channel
	<-c.DriverCh()
	drivers := c.DriversState()
	if drivers[18].TireLapCount != 1 {
		t.Errorf("unexpected lap count - %d", drivers[18].TireLapCount)
	}
	if drivers[18].TireCompound != domain.TireCompoundUnknown {
		t.Errorf("unexpected lap count - %s", drivers[18].TireCompound)
	}
	if drivers[20].TireLapCount != 3 {
		t.Errorf("unexpected lap count - %d", drivers[20].TireLapCount)
	}
	if drivers[20].TireCompound != domain.TireCompoundSoft {
		t.Errorf("unexpected lap count - %s", drivers[20].TireCompound)
	}
	c.Close()
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
