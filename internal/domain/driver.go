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
	Number     string // Number is the unique driver racing number present on their car
	ShortName  string // Shortname is the name abbreviation used on the television broadcast
	Name       string // Name is the full name of the driver
	TeamName   string // TeamName is the short name of the team that the driver races for
	TeamColor  string // TeamColor is the primary color of the team that the driver races for
	TimingData DriverTimingData
}

// Driver domain model represents intrinsic data about a driver as well as updates to live-timing
// data like grid position, gaps, etc.
type DriverTimingData struct {
	// Timing data
	Position    string   // Position is the driver's position on the timing board
	IntervalGap string   // IntervalGap is the time delta between the driver and the driver ahead
	LeaderGap   string   // LeaderGap is the delta between the driver and the lead driver
	LastLap     struct { // Data about the last completed lap
		Time           string // Time is The lap time of the last lap
		IsPersonalBest bool   // PersonalBest indicates if the last lap is a personal best for the driver
	}
	BestLapTime string // BestLapTime is the time of the best lap
	// Stint Data
	TireCompound TireCompound // The current tire compound that the driver is using
	TireLapCount int          // The current lap count that the driver is on
	IsInPit      bool         // InPit indicates if the driver is in the pit
	IsPitOut     bool         // PitOut indicates if the driver is on an outlap
	Retired      bool         // The driver is out of the session due to crash, mechanical failure, etc.
	// Sector times
	Sectors []Sector
	// Qualifying-specific timing data
	BestLapTimes []string // Best times in each session part (applicable for Qualifying sessions only, e.g.: Q1, Q2, Q3)
	KnockedOut   bool     // The driver did not qualify for the current session (only applicable during qualifiying session)
	Cutoff       bool     // The driver is in the cutoff zone (only applicable during qualifiying session)
}

// Sector represents timing data about individual sectors around the lap.
type Sector struct {
	Time           string
	IsPersonalBest bool
	IsOverallBest  bool
	IsActive       bool
}

func NewDriver(number string) Driver {
	return Driver{
		Number: number,
		TimingData: DriverTimingData{
			Sectors:      make([]Sector, 3),
			BestLapTimes: make([]string, 3),
		},
	}
}

/* Driver Intrinsic Data
------------------------------------------------------------------------------------------------- */

func (d *Driver) SetShortName(s *string) {
	if s != nil {
		d.ShortName = *s
	}
}

func (d *Driver) SetName(first *string, last *string, format *string) {
	if first != nil && last != nil {
		if format != nil && *format == "LastNameIsPrimary" {
			d.Name = *last + " " + *first
		} else {
			d.Name = *first + " " + *last
		}
	}
}

func (d *Driver) SetTeamName(s *string) {
	if s != nil {
		d.TeamName = *s
	}
}

func (d *Driver) SetTeamColor(s *string) {
	if s != nil {
		d.TeamColor = "#" + *s
	}
}

/* Driver Timing Data
------------------------------------------------------------------------------------------------- */

func (d *Driver) SetPosition(pos *string) {
	if pos != nil && *pos != "" {
		d.TimingData.Position = *pos
	}
}

func (d *Driver) SetLeaderGap(gap *string) {
	if d.TimingData.Position == "1" {
		d.TimingData.LeaderGap = ""
	} else if gap != nil && *gap != "" {
		d.TimingData.LeaderGap = *gap
	}
}

func (d *Driver) SetIntervalGap(gap *string) {
	if d.TimingData.Position == "1" {
		d.TimingData.IntervalGap = ""
	} else if gap != nil && *gap != "" {
		d.TimingData.IntervalGap = *gap
	}
}

func (d *Driver) SetLastLap(time *string, personalFastest *bool) {
	if time != nil && *time != "" {
		d.TimingData.LastLap.Time = *time
	}

	if personalFastest != nil {
		d.TimingData.LastLap.IsPersonalBest = *personalFastest
	} else {
		d.TimingData.LastLap.IsPersonalBest = false
	}
}

func (d *Driver) SetBestLap(time *string) {
	if time != nil && *time != "" {
		d.TimingData.BestLapTime = *time
	}
}

func (d *Driver) SetKnockedOut(out *bool) {
	if out != nil {
		d.TimingData.KnockedOut = *out
	}
}

func (d *Driver) SetCutoff(cutoff *bool) {
	if cutoff != nil {
		d.TimingData.Cutoff = *cutoff
	}
}

func (d *Driver) SetSector(i int, time *string, personalBest, overallBest *bool) {
	if time != nil {
		d.TimingData.Sectors[i] = Sector{
			IsActive: true,
			Time:     *time,
		}
	}

	if personalBest != nil {
		d.TimingData.Sectors[i].IsPersonalBest = *personalBest
	}

	if overallBest != nil {
		d.TimingData.Sectors[i].IsOverallBest = *overallBest
	}

	if i < 1 {
		d.TimingData.Sectors[1] = Sector{IsActive: false}
	}
	if i < 2 {
		d.TimingData.Sectors[2] = Sector{IsActive: false}
	}
}
