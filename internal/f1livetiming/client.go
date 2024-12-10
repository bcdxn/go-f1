package f1livetiming

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bcdxn/f1cli/internal/domain"
	"github.com/coder/websocket"
)

// NewClient creates and returns a new F1 LiveTiming Client for retrieving real-time data from
// active F1 sessions.
func NewClient(opts ...ClientOption) *Client {
	// get a default client
	c := defaultClient()
	// apply options to the client
	for _, opt := range opts {
		opt(c)
	}
	c.logger.Debug("created new F1 LiveTiming Client")
	return c
}

// Client represents an F1 Live Timing API Client that can connect to the F1 Live Timing API and
// transforms them into a simpler structure that aligns with structures more appropriate for the TUI
type Client struct {
	// channels for consumers to read from
	done     chan error    // a channel to communicate to the outside world that we've closed the websocket
	driverCh chan struct{} // a channel for notifying consumers that driver data has changed
	eventCh  chan struct{} // a channel for notifying consumers that race weekend data has changed
	// internal state to manage async nature of the client
	interrupt chan struct{} // a channel for the outside world to signal to stop listening
	listening bool          // indicates if the websocket connection is alive
	// logger
	logger *slog.Logger
	// Session data
	connectionToken string
	cookie          string
	// F1 Live Timing API configuration
	httpBaseURL string
	wsBaseURL   string
	// Internal state store
	drivers       map[string]domain.Driver
	driversTiming map[string]domain.DriverTimingData
	event         domain.RaceWeekendEvent
}

// Negotiate calls the F1 Live Timing Signalr API, retreiving information required to start the
// websocket connection using the Connect function.
func (c *Client) Negotiate() error {
	c.logger.Debug("negotiating connection")

	if c.connectionToken != "" {
		c.logger.Warn("called Negotiate when connection already negotiated")
		return nil
	}

	req, err := c.negotiateRequest()
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending f1 livetiming api negotiation request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		ct, err := c.parseConnectionToken(resp.Body)
		if err != nil {
			return fmt.Errorf("error parsing connection token: %w", err)
		}
		c.connectionToken = ct
		c.cookie = resp.Header.Get("set-cookie")
		c.logger.Debug("successfully negotiated connection; connection token len:", "token_length", len(ct))
		return nil
	default:
		return fmt.Errorf("error negotiating f1 livetiming api connection: %w", errors.New(resp.Status))
	}
}

// Connect calls the F1 Live Timing API, creating the websocket connection, using values derived
// from the Negotiate call. This websocket is where the client can listen for real-time data about
// an in-progress F1 event.
func (c *Client) Connect() {
	// Ensure negotiate was called before connect
	c.logger.Debug("client.Connect()")
	if c.connectionToken == "" {
		c.logger.Error("client.Negotiate() was not called or a valid connecton token was not returned")
		c.done <- errors.New("missing connection token")
		close(c.done)
		return
	}
	// Drive the websocket URL
	u, err := c.websocketURL()
	if err != nil {
		c.logger.Error("error building websocket URL")
		c.done <- err
		close(c.done)
		return
	}
	// Add required headers
	headers := make(http.Header)
	headers.Add("User-Agent", "BestHTTP")
	headers.Add("Accept-Encoding", "gzip,identity")
	headers.Add("Cookie", c.cookie)
	// Create the websocket connection with the F1 livetiming API server
	conn, _, err := websocket.Dial(context.Background(), u.String(), &websocket.DialOptions{HTTPHeader: headers})
	if err != nil {
		c.logger.Error("error dialing websocket", "err", err.Error())
		c.done <- err
		close(c.done)
		return
	}
	conn.SetReadLimit(-1)
	// Start the subscription by sending a message indicating which messages we're interested in
	err = c.sendSubscribeMsg(conn)
	if err != nil {
		c.done <- err
		close(c.done)
	}
	// Start listening on the websocket connection in a non-blocking go-routine
	closed := make(chan struct{})
	go func() {
		defer close(closed)
		c.listening = true
		for c.listening {
			// listen for messages from the livetiming API on the websocket
			_, msg, err := conn.Read(context.Background())
			if err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
					c.logger.Debug("server closed the websocket connection")
				} else {
					c.logger.Error("error reading websocket", "closeStatus", websocket.CloseStatus(err), "err", err.Error())
				}
				return
			}
			// No errors, process the message from the livetiming API
			c.processMessage(msg)
		}
	}()
	// Wait until an interrupt is received _or_ the websocket connection is closed by the server
	select {
	case <-c.interrupt:
		// Received interrupt; stop listening on the websocket
		c.listening = false
	case <-closed:
		// connection closed by webserver
		c.listening = false
	}
	// ensure we close the websocket properly (whether this is the first part in the close handshake,
	// or the second and final leg of the handshake)
	conn.Close(websocket.StatusNormalClosure, "client shutting down")
	close(c.done)
}

// Close cleans up any open channels and stops the websocket listener if it's still listening.
func (c *Client) Close() {
	if c.listening {
		c.listening = false
		close(c.interrupt)
	}
}

/* Channel Getters
------------------------------------------------------------------------------------------------- */

func (c *Client) DoneCh() <-chan error {
	return c.done
}

// DriverCh returns a readonly version of the driver channel. Read from this channel to be
// notified of internal state changes related to intrinsic driver data.
func (c *Client) DriverCh() <-chan struct{} {
	return c.driverCh
}

// EventCh returns a readonly version of the event channel. Read from this channel to be
// notified of internal state changes related to the race weekend data.
func (c *Client) EventCh() <-chan struct{} {
	return c.eventCh
}

/* State Getters
------------------------------------------------------------------------------------------------- */

// DriversState gets the data within the drivers state store; this holds a snapshot of the intrinsic
// data as well as timing/delta for each driver.
func (c *Client) DriversState() map[string]domain.Driver {
	return c.drivers
}

// EventState gets the data within the event state store; this holds a snapshot of the data
// describing a race weekend.
func (c *Client) EventState() domain.RaceWeekendEvent {
	return c.event
}

/* Optional Function Parameters
------------------------------------------------------------------------------------------------- */

type ClientOption = func(c *Client)

func WithHTTPBaseURL(baseUrl string) ClientOption {
	return func(c *Client) {
		c.httpBaseURL = baseUrl
	}
}

func WithWSBaseURL(baseUrl string) ClientOption {
	return func(c *Client) {
		c.wsBaseURL = baseUrl
	}
}

func WithLogger(l *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = l
	}
}

/* Private types
------------------------------------------------------------------------------------------------- */

// negotiateResponse represents the response body of the F1 Live Timing negotiate API.
type negotiateResponse struct {
	Url                     string  `json:"Url"`
	ConnectionToken         string  `json:"ConnectionToken"`
	ConnectionId            string  `json:"ConnectionId"`
	KeepAliveTimeout        float64 `json:"KeepAliveTimeout"`
	DisconnectTimeout       float64 `json:"DisconnectTimeout"`
	ConnectionTimeout       float64 `json:"ConnectionTimeout"`
	TryWebSockets           bool    `json:"TryWebSockets"`
	ProtocolVersion         string  `json:"ProtocolVersion"`
	TransportConnectTimeout float64 `json:"TransportConnectTimeout"`
	LongPollDelay           float64 `json:"LongPollDelay"`
}

// f1ChangeMessage represents a 'change' message sent on the websocket connection from the server.
// It is a delta between the reference data and any other preceeding change messages.
type f1ChangeMessage struct {
	ChangeSetId string `json:"C"`
	Messages    []struct {
		Hub       string `json:"H"`
		Message   string `json:"M"`
		Arguments []any  `json:"A"`
	} `json:"M"`
}

/* Private Helper Functions
------------------------------------------------------------------------------------------------- */

// defaultClient returns an insance of the client configured with the default logger and base URLs
// pointing at the F1 Live Timing API.
func defaultClient() *Client {
	return &Client{
		interrupt: make(chan struct{}),
		done:      make(chan error),
		driverCh:  make(chan struct{}),
		eventCh:   make(chan struct{}),
		logger:    slog.Default(),
		listening: false,
		// httpBaseURL: "https://livetiming.formula1.com",
		// wsBaseURL:   "wss://livetiming.formula1.com",
		httpBaseURL: "http://localhost:3000",
		wsBaseURL:   "ws://localhost:3000",
		drivers:     make(map[string]domain.Driver),
	}
}

// parseConnectionToken is a helper function that parses the negotiation response and pulls out the
// connectionToken field from the body. This token is required in the subsequent connect request
// that creates the websocket connection.
func (Client) parseConnectionToken(body io.ReadCloser) (string, error) {
	var n negotiateResponse
	var t string

	b, err := io.ReadAll(body)
	if err != nil {
		return t, err
	}

	err = json.Unmarshal(b, &n)
	if err != nil {
		return t, err
	}

	return n.ConnectionToken, nil
}

// negotiateRequest creates the HTTP request object that is required to initiate the connection to
// the F1 Live Timing Signalr API.
func (c Client) negotiateRequest() (*http.Request, error) {
	var r *http.Request
	u, err := url.Parse(c.httpBaseURL)
	if err != nil {
		return r, fmt.Errorf("invalid HTTPBaseURL: %w", err)
	}

	r = &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: u.Scheme,
			Host:   u.Host,
			Path:   "/signalr/negotiate",
			RawQuery: url.Values{
				"connectionData": {`[{"Name":"Streaming"}]`},
				"clientProtocol": {"1.5"},
			}.Encode(),
		},
	}

	return r, nil
}

// websocketURL is a helper method that generates the URL with appropriate query parameters
// required to start the websocket connection.
func (c Client) websocketURL() (*url.URL, error) {
	var u *url.URL
	u, err := url.Parse(c.wsBaseURL)
	if err != nil {
		return u, fmt.Errorf("invalid HTTPBaseURL: %w", err)
	}

	u = &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   "/signalr/connect",
		RawQuery: url.Values{
			"connectionData":  {`[{"Name":"Streaming"}]`},
			"connectionToken": {c.connectionToken},
			"clientProtocol":  {"1.5"},
			"transport":       {"webSockets"},
		}.Encode(),
	}

	return u, nil
}

// sendSubscribeMsg sends a message that tells the server which types of data messages we would like
// to receive as required by the F1 Live Timing API.
func (Client) sendSubscribeMsg(conn *websocket.Conn) error {
	return conn.Write(context.Background(), websocket.MessageText, []byte(`
			{
					"H": "Streaming",
					"M": "Subscribe",
					"A": [[
							"Heartbeat",
							"TimingStats",
							"TimingAppData",
							"TrackStatus",
							"DriverList",
							"RaceControlMessages",
							"SessionInfo",
							"SessionData",
							"LapCount",
							"TimingData"
					]],
					"I": 1
			}
	`))
}

// atoui8 converts a string to an unit8.
func (c Client) atoui8(numberStr string) uint8 {
	numberInt, err := strconv.Atoi(numberStr)
	if err != nil {
		c.logger.Warn("invalid number as driver info key", "driverNumber", numberInt)
	}
	return uint8(numberInt)
}

var (
	// The F1 API returns a mixed-type array which makes ummarshalling to strongly typed structs
	// challenging, so we just strip the offending property from the message string using the kfRe
	// regular expression.
	kfRe = regexp.MustCompile(`,\s*"_kf":\s*(?:true|false)`)
)

// processMessage checks the message coming the F1 LiveTiming Client to see if it is a 'change'
// message or a 'reference' message and handles them appropriately, transforming the message into
// 1 to none or many messages that can be written to the client channels.
func (c *Client) processMessage(msg []byte) {
	// Always try to parse a change message first since there is only 1 reference message and
	// tens of thousands of change messages over the course of a session
	var changeData f1ChangeMessage
	err := json.Unmarshal(msg, &changeData)
	if err == nil && len(changeData.ChangeSetId) > 0 && len(changeData.Messages) > 0 {
		c.logger.Debug("received change data message")
		c.processChangeMessage(changeData)
		return
	}
	// Next try to parse a reference data message
	referenceMsg := kfRe.ReplaceAllString(string(msg), "")
	var referenceData f1ReferenceMessage
	err = json.Unmarshal([]byte(referenceMsg), &referenceData)
	if err == nil && referenceData.MessageInterval != "" {
		c.logger.Debug("received reference data message")
		c.logger.Debug("REFERENCE MESSAGE")
		c.logger.Debug(string(msg))
		c.processReferenceMessage(referenceData)
		return
	}
	// The message wasn't a known 'change' or 'reference' message type
	c.logger.Debug("unhandled message", "msg", msg)
}

// processReferenceMessage deconstructs the reference messagef rom the F1 LiveTiming API that
// represents an intitial state, and passes the various components to individual functions that
// handle updating the client's internal state.
func (c *Client) processReferenceMessage(referenceMessage f1ReferenceMessage) {
	c.updateSessionInfo(referenceMessage.Reference.SessionInfo)
	c.updateDriverIntrinsicData(referenceMessage.Reference.DriverList)
	c.updateLapCountData(referenceMessage.Reference.LapCount)
	c.updateDriverTimingData(
		changeTimingDataFromReference(referenceMessage.Reference.TimingData),
	)
	c.updateTimingAppData(
		changeTimingAppDataFromReference(referenceMessage.Reference.TimingAppData),
	)
}

// processChangeMessage handles an incoming change message from the F1 Live Timing API; change
// messages represent deltas to the original reference message and all preceeding change messages.
// Once processed, a simplified event is emitted for use by the TUI.
func (c *Client) processChangeMessage(changeMessage f1ChangeMessage) {
	for _, m := range changeMessage.Messages {
		if m.Hub == "Streaming" && m.Message == "feed" && len(m.Arguments) == 3 {
			msgType := m.Arguments[0]
			msgData := m.Arguments[1]
			// Marshal the data part of the message so that we can unmarshal into strongly typed messages
			// based on the messageType value.
			msg, err := json.Marshal(msgData)
			if err != nil {
				c.logger.Warn("unable to marshal msg", "msg", msg)
				return
			}
			switch msgType {
			case "DriverList":
				c.handleDriverListMsg(msg)
			case "TimingData":
				c.handleDriverTimingData(msg)
			case "SessionInfo":
				c.handleSessionInfoMsg(msg)
			case "LapCount":
				c.handleLapCountMsg(msg)
			case "TimingAppData":
				c.handleTimingAppDataMsg(msg)
			default:
				c.logger.Warn("unknown change message", "type", msgType, "msg", msg)
			}
		}
	}
}

// notifyEventChannel notifies consumers that the client race weekend event state has changed.
func (c *Client) notifyEventChannel() {
	c.logger.Debug("event state change", "event", c.event)
	c.eventCh <- struct{}{}
}

// notifyDriverChannel notifies consumers that the client drivers state has changed.
func (c *Client) notifyDriverChannel() {
	c.logger.Debug("drivers state change", "drivers", c.drivers)
	c.driverCh <- struct{}{}
}

/* Update Internal State
------------------------------------------------------------------------------------------------- */

const (
	f1APIDateLayout = "2006-01-02T15:04:05-0700" // date format used by the F1 LiveTiming API
)

// handleDriverListMsg converts the websocket message to a strongly typed struct and updates the
// client's store via `updateDriverIntrinsicData`.
func (c *Client) handleDriverListMsg(msg []byte) {
	var driverDataMsg map[string]driverData
	err := json.Unmarshal(msg, &driverDataMsg)
	if err != nil {
		c.logger.Warn("driver data msg in unknown format", "msg", string(msg))
		return
	}

	c.updateDriverIntrinsicData(driverDataMsg)
}

// updateDriverIntrinsicData converts DriverList msg from the F1 Live Timing API to the Driver
// domain models stored in the client's internal state store and writes a notification even to the
// driver channel to let consumers know that the data has been updated.
func (c *Client) updateDriverIntrinsicData(driverDataMsg map[string]driverData) {
	// update data for each driver to the drivers map
	for number, driverData := range driverDataMsg {
		// retrieve existing driver data from the map if it exists or create a new driver
		driver, ok := c.drivers[number]
		if !ok {
			driver = domain.NewDriver(number)
		}
		// Overwrite fields
		driver.SetShortName(driverData.ShortName)
		driver.SetName(driverData.FirstName, driverData.LastName, driverData.NameFormat)
		driver.SetTeamName(driverData.TeamName)
		driver.SetTeamColor(driverData.TeamColour)
		// write the driver data back to the client state store
		c.drivers[number] = driver
	}
	c.notifyDriverChannel()
}

// handleDriverTimingData converts the TimingData websocket message from the F1 LiveTiming API to
// strongly typed struct and updates the client's stre via the `updateDriverTimingData`.
func (c *Client) handleDriverTimingData(msg []byte) {
	var timingDataMsg changeTimingData
	err := json.Unmarshal(msg, &timingDataMsg)
	if err != nil {
		c.logger.Warn("timing data msg in unknown format", "msg", string(msg))
		return
	}
	c.updateDriverTimingData(timingDataMsg)
}

// updateDriverTimingData converts TimingData msg from the F1 Live Timing API to the Driver domain
// models stored in the client's internal state store and writes a notification even to the driver
// channel to let consumers know that the data has been updated.
func (c *Client) updateDriverTimingData(timingDataMsg changeTimingData) {
	// only send a notification event fon the session channel if the session was updated
	sessionUpdated := false
	// add data for each driver to the drivers map
	for number, timingData := range timingDataMsg.Lines {
		// retrieve existing driver data from the map if it exists or create a new driver
		driver, ok := c.drivers[number]
		if !ok {
			driver = domain.NewDriver(number)
		}
		// Overwrite fields
		driver.SetPosition(timingData.Position)
		driver.SetLeaderGap(timingData.GapToLeader)
		driver.SetIntervalGap(timingData.IntervalToPositionAhead.Value)
		driver.SetLastLap(timingData.LastLapTime.Value, timingData.LastLapTime.PersonalFastest)
		if timingData.LastLapTime.OverallFastest != nil && *timingData.LastLapTime.OverallFastest {
			c.event.Session.FastestLapOwner = number
			sessionUpdated = true
		}
		driver.SetBestLap(timingData.BestLapTime.Value)
		driver.SetKnockedOut(timingData.KnockedOut)
		driver.SetCutoff(timingData.Cutoff)
		// In Qualifying Sessions the interval is stored separately for each qualifying part; we're only
		// interested the most recent qualifying part, so we iterate through (the list is in order) and
		// overwrite the gaps for each available qualifying part
		parts := make([]string, 0, 3)
		for part := range timingData.Stats {
			parts = append(parts, part)
		}
		sort.Strings(parts)
		for _, part := range parts {
			driver.SetLeaderGap(timingData.Stats[part].TimeDiffToFastest)
			driver.SetIntervalGap(timingData.Stats[part].TimeDiffToPositionAhead)
		}

		// Sort sectors
		sectorNums := make([]string, 0, 3)
		for sectorNum := range timingData.Sectors {
			sectorNums = append(sectorNums, sectorNum)
		}
		sort.Strings(sectorNums)
		for _, sectorNum := range sectorNums {
			i, _ := strconv.Atoi(sectorNum)
			sector := timingData.Sectors[sectorNum]
			driver.SetSector(i, sector.Value, sector.PersonalBest, sector.OverallBest)
			if sector.OverallBest != nil && *sector.OverallBest {
				c.event.Session.FastestSectorOwner[i] = number
				sessionUpdated = true
			}
		}
		c.drivers[number] = driver
	}

	c.notifyDriverChannel()
	if sessionUpdated {
		c.notifyEventChannel()
	}
}

// handleSessionInfoMsg converts websocket msg to strongly typed struct and update's the client
// store via the `updateSessionInfo` helper function.
func (c *Client) handleSessionInfoMsg(msg []byte) {
	var s sessionInfo
	err := json.Unmarshal(msg, &s)
	if err != nil {
		c.logger.Warn("timing data msg in unknown format", "msg", string(msg))
		return
	}

	c.updateSessionInfo(s)
}

// updateSessionInfo converts SessionInfo msg from the F1 Live Timing API to the `RaceWeekendEvent`
// stored in the client's internal state store and writes a notification on the event channel to let
// consumers know the data has changed.
func (c *Client) updateSessionInfo(session sessionInfo) {
	if session.Meeting.Name != nil {
		c.event.Name = *session.Meeting.Name
	}
	if session.Meeting.OfficialName != nil {
		c.event.FullName = *session.Meeting.OfficialName
	}
	if session.Meeting.Location != nil {
		c.event.Location = *session.Meeting.Location
	}
	if session.Meeting.Number != nil {
		c.event.RoundNumber = *session.Meeting.Number
	}
	if session.Meeting.Country.Code != nil {
		c.event.CountryCode = *session.Meeting.Country.Code
	}
	if session.Meeting.Country.Name != nil {
		c.event.CountryName = *session.Meeting.Country.Name
	}
	if session.Meeting.Circuit.ShortName != nil {
		c.event.CircuitShortName = *session.Meeting.Circuit.ShortName
	}
	if session.Name != nil {
		c.event.Session.Name = *session.Name
	}
	if session.GMTOffset != nil {
		c.event.Session.GMTOffset = strings.Join(strings.Split(*session.GMTOffset, ":")[:2], "")
	}
	if session.StartDate != nil && c.event.Session.GMTOffset != "" {
		c.event.Session.StartDate, _ = time.Parse(f1APIDateLayout, *session.StartDate+c.event.Session.GMTOffset)
	}
	if session.EndDate != nil && c.event.Session.GMTOffset != "" {
		c.event.Session.EndDate, _ = time.Parse(f1APIDateLayout, *session.EndDate+c.event.Session.GMTOffset)
	}
	c.event.Session.FastestSectorOwner = make([]string, 3)
	c.notifyEventChannel()
}

// handleLapCountMsg converts a websocket message from the F1 Live Timing API to a strongly typed
// struct and updates the client's internal store via the `updateLapCountData` helper function.
func (c *Client) handleLapCountMsg(msg []byte) {
	var lc lapCount
	err := json.Unmarshal(msg, &lc)
	if err != nil {
		c.logger.Warn("lap count msg in unknown format", "msg", string(msg))
		return
	}

	c.updateLapCountData(lc)
}

// updateLapCountData converts LapCount msg from the F1 Live Timing API to the RaceWeekendEvent`
// stored in the client's internal state store and writes a notification on the event channel to let
// consumers know the data has changed.
func (c *Client) updateLapCountData(lc lapCount) {
	if lc.CurrentLap != nil {
		c.event.Session.CurrentLap = *lc.CurrentLap
	}
	if lc.TotalLaps != nil {
		c.event.Session.TotalLaps = *lc.TotalLaps
	}

	c.notifyEventChannel()
}

// handleTimingAppDataMsg converts the websocket message from the F1 Live Timing API to a strongly
// typed struct and then updates the client's internal store via the `updateTimingAppData` helper
// function.
func (c *Client) handleTimingAppDataMsg(msg []byte) {
	var tad changeTimingAppData
	err := json.Unmarshal(msg, &tad)
	if err != nil {
		c.logger.Warn("timing app data msg in unknown format", "msg", string(msg))
		return
	}
}

// updateTimingAppData converts TimingAppData msg from the F1 Live Timing API to the
// `Driver` domain type stored in the client's internal state store and writes a notification on the
// driver channel to let consumers know the data has changed.
func (c *Client) updateTimingAppData(tad changeTimingAppData) {
	// for driverNumber, timingAppData := range tad.Lines {
	// 	number := c.atoui8(driverNumber)
	// 	if number == 0 {
	// 		continue
	// 	}
	// 	// if multiple stints are given (e.g. in the reference message) we'll iterate through them,
	// 	// taking the stint with the largest key (which are numbers indicated the order)
	// 	stints := make([]string, 0, 3)
	// 	for stint := range timingAppData.Stints {
	// 		stints = append(stints, stint)
	// 	}
	// 	// sort the stints in descending order by key so we can take the largest key at index 0
	// 	sort.Slice(stints, func(i, j int) bool {
	// 		return stints[i] > stints[j]
	// 	})
	// 	if len(stints) == 0 {
	// 		continue
	// 	}
	// 	currentStint := stints[0]

	// 	driver, ok := c.drivers[number]
	// 	if !ok {
	// 		driver = domain.Driver{Number: number}
	// 	}
	// 	if timingAppData.Stints[currentStint].Compound != nil {
	// 		driver.TireCompound = tireCompound(*timingAppData.Stints[currentStint].Compound)
	// 	}
	// 	if timingAppData.Stints[currentStint].TotalLaps != nil {
	// 		driver.TireLapCount = *timingAppData.Stints[currentStint].TotalLaps
	// 	}
	// 	// overwrite the driver state with the new stint information
	// 	c.drivers[number] = driver
	// }

	c.notifyDriverChannel()
}

func tireCompound(compound string) domain.TireCompound {
	switch compound {
	case "SOFT":
		return domain.TireCompoundSoft
	case "MEDIUM":
		return domain.TireCompoundMedium
	case "HARD":
		return domain.TireCompoundHard
	case "INTERMEDIATE":
		return domain.TireCompoundIntermediate
	case "WET":
		return domain.TireCompoundFullWet
	default:
		return domain.TireCompoundUnknown
	}
}
