package domain

// Driver domain events represent intrinsic data about a driver as well as updates to live-timing
// data like grid position, gaps, etc.
type Driver struct {
	Number       int
	ShortName    string
	Name         string
	TeamName     string
	TeamColor    string
	Position     int
	IntervalGap  string
	LeaderGap    string
	TireCompound string
	TireLapCount int
	LastLapTime  string
	BestLapTime  string
	InPit        *bool
	PitOut       *bool
}
