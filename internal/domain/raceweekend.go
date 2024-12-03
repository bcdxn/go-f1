package domain

import "time"

type SessionType string

const (
	SessionTypeTest       SessionType = "Test"
	SessionTypePractice   SessionType = "Practice"
	SessionTypeQualifying SessionType = "Qualifying"
	SessionTypeRace       SessionType = "Race"
)

// Meeting represents data about the race weekend event. This data applies to all of the sessions
// within a race weekend.
type RaceWeekendEvent struct {
	Name             string  // Name is the informal name of the race weekend event
	FullName         string  // FullName is the full official name of the event including primary sponsor
	Location         string  // Location is the locality in which the race weekend is taking place
	RoundNumber      uint8   // The sequence number of the race weekend event within the season
	CountryCode      string  // The 2-3 letter code indicating the country in which the event is taking place
	CountryName      string  // The full name of the country in which the event is taking place
	CircuitShortName string  // The informal name of the circuit at which the event is taking place
	Session          Session // A Race Weekend is composed of multiple sessions; only the active session is represented
}

// Session represents the various sessions within a race weekend, e.g.:
type Session struct {
	Type            SessionType
	Number          uint8     // For sessions that have multiple rounds, e.g.: practice *2* and qualifying *3*
	Name            string    // The name of the session, e.g.: "Practice 1", "Race", etc.
	StartDate       time.Time // The start of the session
	EndDate         time.Time // The end time of the session - will be zerovalue until session has ended
	GMTOffset       string    // GMTOffset is the track-timezone delta with GMT/UTC
	FastestLapOwner uint8     // FastestLapOwner is the number of the driver that has the fastest lap in the session
	CurrentLap      uint8     // The current lead lap (only applicable for races)
	TotalLaps       uint8     // The total number of planned laps (only applicable for races)
}
