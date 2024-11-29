package f1livetiming

import "time"

// f1ReferenceMessance represents the initial state of a session for all of the requested data from
// the F1 Live Timing API. This includes intrinsic data about the session as well as driver, timing
// and status data. The reference message should be used to create an initial state; all other
// messages are 'Change' data messages that alter the state managed by the API consumer.
type f1ReferenceMessage struct {
	Reference struct {
		Heartbeat     heartbeat              `json:"Heartbeat"`           // Heartbeat is the most recent heartbeat emitted
		TimingAppData referenceTimingAppData `json:"TimingAppData"`       // TimingAppData contains per-driver stint information
		DriverList    map[string]driverData  `json:"DriverList"`          // DriverList contains per-driver intrinsic data
		RaceCtrlMsgs  referenceRaceCtrlMsgs  `json:"RaceControlMessages"` // RaceCtrlMsgs contains all emitted race control messages
		SessionInfo   sessionInfo            `json:"SessionInfo"`         // SessionInfo contains intrinsic data about the event and session
		SessionData   referenceSessionData   `json:"SessionData"`         // SesionData contains all emitted session and track status changes
		TimingData    referenceTimingData    `json:"TimingData"`          // TimingData represents driver-specific lap times, intervals, etc.
		LapCount      lapCount               `json:"LapCount"`            // LapCount contains the latest lap (current/total) data
	} `json:"R"`
	MessageInterval string `json:"I"`
}

// The heartbeat message indicates the client connection to the server is working even if there are
// no other messages coming from the server.
type heartbeat struct {
	ReceivedAt time.Time `json:"Utc"`
}

// referenceTimingAppData contains per-driver stint information including lap count and tire
// compound.
type referenceTimingAppData struct {
	Lines map[string]struct {
		RacingNumber string  `json:"RacingNumber"`
		Line         int     `json:"Line"`
		GridPos      string  `json:"GridPos"`
		Stints       []stint `json:"Stints"`
	} `json:"Lines"`
}

// changeTimingAppData contains per-driver stint information similar to `referenceTimingAppData` but
// the data structure containing the stint information is a map instead of a slice.
type changeTimingAppData struct {
	Lines map[string]struct {
		RacingNumber string           `json:"RacingNumber"`
		Line         int              `json:"Line"`
		GridPos      string           `json:"GridPos"`
		Stints       map[string]stint `json:"Stints"`
	} `json:"Lines"`
}

type stint struct {
	LapFlags        int    `json:"LapFlags"`
	Compound        string `json:"Compound"`
	New             string `json:"New"`
	TyresNotChanged string `json:"TyresNotChanged"`
	TotalLaps       int    `json:"TotalLaps"`
	StartLaps       int    `json:"StartLaps"`
	LapTime         string `json:"LapTime"`
	LapNumber       int    `json:"LapNumber"`
}

type trackStatus struct {
	Status  string `json:"Status"`
	Message string `json:"Message"`
}

// driverData represents intrinsic data about an individual driver
type driverData struct {
	RacingNumber  string `json:"RacingNumber"`
	BroadcastName string `json:"BroadcastName"`
	FullName      string `json:"FullName"`
	ShortName     string `json:"Tla"`
	Line          int    `json:"Line"`
	TeamName      string `json:"TeamName"`
	TeamColour    string `json:"TeamColour"`
	FirstName     string `json:"FirstName"`
	LastName      string `json:"LastName"`
	Reference     string `json:"Reference"`
	CountryCode   string `json:"CountryCode"`
	HeadshotURL   string `json:"HeadshotUrl"`
}

// changeRaceCtrlMsgs contains a map of race control messages.
type changeRaceCtrlMsgs struct {
	Messages map[string]raceCtrlMsg `json:"Messages"`
}

// referenceRaceCtrlMsgs contains a list of race control messages.
type referenceRaceCtrlMsgs struct {
	Messages []raceCtrlMsg `json:"Messages"`
}

// raceCtrlMsgs represents a message or alert issued by Race Control. This includes information
// about investigations, penalties, track limits violations, flag information and more.
type raceCtrlMsg struct {
	UTC      string `json:"Utc"`
	Lap      uint8  `json:"Lap"`
	Category string `json:"Category"`
	Message  string `json:"Message"`
	Flag     string `json:"Flag"`
	Mode     string `json:"Mode"`
	Scope    string `json:"Scope"`
	Status   string `json:"Status"`
	Sector   uint8  `json:"Sector"`
}

// sessionInfo contains intrinsic data about the weekend event and current session. Typically this
// event is consumed as a part of the initial reference message without significant changes
// throughout the session
type sessionInfo struct {
	Meeting struct {
		Key          int    `json:"Key"`
		Name         string `json:"Name"`
		OfficialName string `json:"OfficialName"`
		Location     string `json:"Location"`
		Number       int    `json:"Number"`
		Country      struct {
			Key  int    `json:"Key"`
			Code string `json:"Code"`
			Name string `json:"Name"`
		} `json:"Country"`
		Circuit struct {
			Key       int    `json:"Key"`
			ShortName string `json:"ShortName"`
		} `json:"Circuit"`
	} `json:"Meeting"`
	ArchiveStatus struct {
		Status string `json:"Status"`
	} `json:"ArchiveStatus"`
	Key       int    `json:"Key"`
	Type      string `json:"Type"`
	Number    int    `json:"Number"`
	Name      string `json:"Name"`
	StartDate string `json:"StartDate"`
	EndDate   string `json:"EndDate"`
	GmtOffset string `json:"GmtOffset"`
	Path      string `json:"Path"`
}

// referenceSessionData contains a list of all session/track status changes and the corresponding
// lap in which the changes occurred (if the session is a race).
type referenceSessionData struct {
	Series       []sessionDataSeries       `json:"Series"` // Lap on which the status applies
	StatusSeries []sessionDataStatusSeries `json:"StatusSeries"`
}

// changeSessionData contains a map of all sessin/track status changes and the corresponding lap in
// which the changes occurred (if the session is  race). It is identical to the
// `referenceSessiondata` type except that the changes are represented in a map as opposed to a list
// so that changes can be easily merged into a state store.
type changeSessionData struct {
	Series       map[string]sessionDataSeries       `json:"Series"`
	StatusSeries map[string]sessionDataStatusSeries `json:"StatusSeries"`
}

// sessionDataSeries contains the lap count for which the session data status applies. e.g. a Yellow
// flag on Lap 1 of the race. Note: this will only apply for races and sprint races.
type sessionDataSeries struct {
	UTC time.Time `json:"Utc"`
	Lap uint8     `json:"Lap"`
}

// sessionDataStatuseries contains a session and/or track status series change. These statuses
// include flags, (virtual) safety cards, etc.
type sessionDataStatusSeries struct {
	Utc           time.Time `json:"Utc"`
	TrackStatus   string    `json:"TrackStatus"`
	SessionStatus string    `json:"SessionStatus"`
}

// referenceTimingData represents per-driver live timing data including lap times, gaps, personal/
// overall best indicators and sector timing data. The only difference between `referenceTimingData`
// and `changeTimingData` is that sector data is represented as a list in `referenceTimingData`.
type referenceTimingData struct {
	Lines map[string]referenceDriverTimingData `json:"Lines"`
}

// changeTimingData represents per-driver live timing data including lap times, gaps, personal/
// overall best indicators and sector timing data. The only difference between `changeTimingData`
// and `referenceTimingData` is that sector data is represented as a map in `changeTimingData` so
// that changes can be easily merged in a state store.
type changeTimingData struct {
	Lines map[string]changeDriverTimingData `json:"Lines"`
}

// driverTimingData contains lap times, gaps and other live-timing information about a specific
// driver. Both `referenceDriverTimingData` and `changeDriverTimingData` 'inherit' the properties
// from `driverTimingData`
type driverTimingData struct {
	TimeDiffToFastest       string `json:"TimeDiffToFastest"`
	TimeDiffToPositionAhead string `json:"TimeDiffToPositionAhead"`
	Line                    int    `json:"Line"`         // current position
	Position                string `json:"Position"`     // current position
	ShowPosition            bool   `json:"ShowPosition"` // Will be false when a driver is out of the session (race), or out of the session (qualifying)
	RacingNumber            string `json:"RacingNumber"` // the unique driver number
	Retired                 bool   `json:"Retired"`      // car and driver have retired from the race
	InPit                   bool   `json:"InPit"`        // car is in pit
	PitOut                  bool   `json:"PitOut"`       // current lap is an out-lap
	Stopped                 bool   `json:"Stopped"`      // true when car is not moving
	Status                  int    `json:"Status"`
	GapToLeader             string `json:"GapToLeader"`
	IntervalToPositionAhead struct {
		Value    string `json:"Value"`
		Catching bool   `json:"Catching"`
	} `json:"IntervalToPositionAhead"`
	Speeds struct {
		FirstIntermediatePoint  driverSpeedTimingData `json:"I1"`
		SecondIntermediatePoint driverSpeedTimingData `json:"I2"`
		SpeedTrap               driverSpeedTimingData `json:"ST"`
	} `json:"Speeds"`
	BestLapTime struct {
		Value string `json:"Value"`
		Lap   int    `json:"Lap"`
	} `json:"BestLapTime"`
	LastLapTime struct {
		Value           string `json:"Value"`
		Status          int    `json:"Status"`
		OverallFastest  bool   `json:"OverallFastest"`
		PersonalFastest bool   `json:"PersonalFastest"`
	} `json:"LastLapTime"`
	NumberOfLaps int `json:"NumberOfLaps"`
}

// driverSpeedTimingData represents speed-trap-like data captured at various points around the
// circuit for a specific driver on a particular lap.
type driverSpeedTimingData struct {
	Value           string `json:"Value"`
	Status          int    `json:"Status"`
	OverallFastest  bool   `json:"OverallFastest"`
	PersonalFastest bool   `json:"PersonalFastest"`
}

// referenceDriverTimingData contains driver timing data along with sector timing data in a slice.
type referenceDriverTimingData struct {
	driverTimingData
	Sectors []sectorTiming `json:"Sectors"`
}

// changeDriverTimingData contains driver timing data along with sector timing data in a map so
// that changes can be easily merged into a data store.
type changeDriverTimingData struct {
	driverTimingData
	Sectors map[string]sectorTiming `json:"Sectors"`
}

// sectorTiming represents timing for 1 of 3 sectors around the crcuit for a specific driver on a
// particular lap.
type sectorTiming struct {
	Stopped         bool   `json:"Stopped"`
	Value           string `json:"Value"`
	Status          int    `json:"Status"`
	OverallFastest  bool   `json:"OverallFastest"`
	PersonalFastest bool   `json:"PersonalFastest"`
	Segments        []struct {
		Status int `json:"Status"`
	} `json:"Segments"`
	PreviousValue string `json:"PreviousValue"`
}

// lapCount represents the latest lap information of the session, in cluding the `CurrentLap` of the
// leader in races.`
type lapCount struct {
	CurrentLap uint8 `json:"CurrentLap"`
	TotalLaps  uint8 `json:"TotalLaps"`
}
