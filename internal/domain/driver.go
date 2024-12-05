package domain

type TireCompound string

const (
	TireCompoundSoft         TireCompound = "Soft"
	TireCompoundMedium       TireCompound = "Medium"
	TireCompoundHard         TireCompound = "Hard"
	TireCompoundIntermediate TireCompound = "Intermediate"
	TireCompoundFullWet      TireCompound = "Wet"
	TireCompoundUnknown      TireCompound = "Unknown"
)

// Driver domain model represent intrinsic data about a driver as well as updates to live-timing
// data like grid position, gaps, etc.
type Driver struct {
	// Intrinsic Data
	Number    uint8  // Number is the unique driver racing number present on their car
	ShortName string // Shortname is the name abbreviation used on the television broadcast
	Name      string // Name is the full name of the driver
	TeamName  string // TeamName is the short name of the team that the driver races for
	TeamColor string // TeamColor is the primary color of the team that the driver races for
	// Timing Data
	Position    int      // Position is the driver's order on track or timing order depending on session type
	IntervalGap string   // IntervalGap is the time delta between the driver and the driver ahead
	LeaderGap   string   // LeaderGap is the delta between the driver and the lead driver
	LastLap     struct { // Data about the last completed lap
		Time           string // Time is The lap time of the last lap
		IsPersonalBest bool   // PersonalBest indicates if the last lap is a personal best for the driver
	}
	BestLapTime string // BestLapTime is the time of the best lap
	// Stint Data
	TireCompound TireCompound // The current tire compound that the driver is using
	TireLapCount uint8        // The current lap count that the driver is on
	IsInPit      bool         // InPit indicates if the driver is in the pit
	IsPitOut     bool         // PitOut indicates if the driver is on an outlap
	OutOfSession bool         // The driver is out of the session due to crash, mechanical failure, etc.
}
