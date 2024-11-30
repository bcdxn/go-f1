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
	"strconv"

	"github.com/bcdxn/f1cli/internal/domain"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
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
	// channels
	Interrupt chan struct{}
	DriverCh  chan domain.Driver
	// logger
	logger *slog.Logger
	// Session data
	connectionToken string
	cookie          string
	// F1 Live Timing API configuration
	httpBaseURL string
	wsBaseURL   string
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
func (c *Client) Connect() error {
	if c.connectionToken == "" {
		return errors.New("client.Negotiate() was not called or a valid connecton token was not returned")
	}

	u, err := c.websocketURL()
	if err != nil {
		return err
	}

	conn, _, err := websocket.Dial(context.Background(), u.String(), nil)
	if err != nil {
		return err
	}
	defer conn.CloseNow()
	// Start the subscription by sending a message indicating which messages we're interested in
	c.sendSubscribeMsg(conn)

	listening := true

	go func() {
		for listening {
			_, msg, err := conn.Read(context.Background())
			if err != nil {
				c.logger.Error("error reading websocket", "err", err.Error())
				listening = false
			}

			c.processMessage(msg)
		}
	}()

	return nil
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
		logger:      slog.Default(),
		Interrupt:   make(chan struct{}),
		httpBaseURL: "https://livetiming.formula1.com",
		wsBaseURL:   "wss://livetiming.formula1.com",
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
	return wsjson.Write(context.Background(), conn, `
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
    `)
}

var (
	// The F1 API returns a mixed-type array which makes ummarshalling challenging, so we just strip
	// the offending property from the message string using the kfRe regular expression.
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
		// c.processReferenceMessage(referenceData)
		return
	}
	// The message wasn't a known 'change' or 'reference' message type
	c.logger.Debug("unhandled message", "msg", msg)
}

// processChangeMessage handles an incoming change message from the F1 Live Timing API; change
// messages represent deltas to the original reference message and all preceeding change messages.
// Once processed, a simplified event is emitted for use by the TUI.
func (c *Client) processChangeMessage(changeMessage f1ChangeMessage) {
	for _, m := range changeMessage.Messages {
		if m.Hub == "Streaming" && m.Message == "feed" && len(m.Arguments) == 3 {
			msgType := m.Arguments[0]
			msgData := m.Arguments[1]
			switch msgType {
			case "DriverList":
				c.writeDriverListToDriverChannel(msgData)
			default:
				c.logger.Warn("unknown change message", "type", msgType, "msg", msgData)
			}
		}
	}
}

/* Channel Senders
------------------------------------------------------------------------------------------------- */

// writeDriverListToDriverChannel converts DriverList data from the F1 Live Timing API to the Driver
// domain model and emits it on the Driver channel
func (c *Client) writeDriverListToDriverChannel(msg any) {
	str, err := json.Marshal(msg)
	if err != nil {
		c.logger.Warn("unable to serialize msg", "msg", msg)
		return
	}

	var driverDataMsg map[string]driverData
	err = json.Unmarshal(str, &driverDataMsg)
	if err != nil {
		c.logger.Warn("driver data msg in unknown format", "msg", str)
		return
	}

	// emit driver data for each driver
	for driverNumber, driverData := range driverDataMsg {
		number, err := strconv.Atoi(driverNumber)
		if err != nil {
			c.logger.Warn("invalid driver number as driver info key", "driverNumber", driverNumber)
			continue
		}
		c.DriverCh <- domain.Driver{
			Number:    number,
			ShortName: driverData.ShortName,
			Name:      driverData.FullName,
			Position:  driverData.Line,
			TeamName:  driverData.TeamName,
			TeamColor: driverData.TeamColour,
		}
	}
}
