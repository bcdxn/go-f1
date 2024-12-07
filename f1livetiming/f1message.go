package f1livetiming

import "time"

type F1ReferenceMessage struct {
	Reference struct {
		Heartbeat           HeartBeat              `json:"Heartbeat"`
		CarDataZ            string                 `json:"CarData.z"`
		PositionDataZ       string                 `json:"Position.z"`
		ExtrapolatedClock   ExtrapolatedClock      `json:"ExtrapolatedClock"`
		TopThree            TopThree               `json:"TopThree"`
		TimingStats         TimingStats            `json:"TimingStats"`
		TimingAppData       ReferenceTimingAppData `json:"TimingAppData"`
		WeatherData         WeatherData            `json:"WeatherData"`
		DriverList          map[string]DriverData  `json:"DriverList"`
		RaceControlMessages RaceControlMessages    `json:"RaceControlMessages"`
		SessionInfo         SessionInfo            `json:"SessionInfo"`
		SessionData         ReferenceSessionData   `json:"SessionData"`
		TimingData          TimingData             `json:"TimingData"`
		LapCount            LapCount               `json:"LapCount"`
	} `json:"R"`
	MessageInterval string `json:"I"`
}

type HeartBeat struct {
	UTC time.Time `json:"Utc"`
}

type CarDataMessage struct {
	Entries []struct {
		UTC  time.Time `json:"Utc"`
		Cars map[string]struct {
			Channels IndividualCarData `json:"Channels"`
		} `json:"Cars"`
	} `json:"Entries"`
}

type IndividualCarData struct {
	// Engine revolutions per minute ranges from 0 - 15000
	RPM int `json:"0"`
	// Speed is in KPH
	Speed int `json:"2"`
	// Gears from 0 - 8 where 0 indicates neutral
	Gear int `json:"3"`
	// Indicates if the break is pressed 0 or 104
	Break int `json:"4"`
	// Percent throttle ranges from 0 - 104
	Throttle int `json:"5"`
	// | API Value | Meaning |
	// |:----------|:--------|
	// | 0         | DRSoff  |
	// | 1         | DRS off |
	// | 2         | ?       |
	// | 3         | ?       |
	// | 8         | Detected, eligible once in activation zone |
	// | 9         | ?       |
	// | 10        | DRS on  |
	// | 12        | DRS on  |
	// | 14        | DRS on  |
	HasDRS int `json:"45"`
}

type ExtrapolatedClock struct {
	UTC           time.Time `json:"Utc"`
	Remaining     string    `json:"Remaining"`
	Extrapolating bool      `json:"Extrapolating"`
}

type TopThree struct {
	Withheld bool `json:"Withheld"`
	Lines    []struct {
		Position        string `json:"Position"`
		ShowPosition    bool   `json:"ShowPosition"`
		RacingNumber    string `json:"RacingNumber"`
		ShortName       string `json:"Tla"`
		BroadcastName   string `json:"BroadcastName"`
		FullName        string `json:"FullName"`
		LapTime         string `json:"LapTime"`
		LapState        int    `json:"LapState"`
		DiffToAhead     string `json:"DiffToAhead"`
		DiffToLeader    string `json:"DiffToLeader"`
		OverallFastest  bool   `json:"OverallFastest"`
		PersonalFastest bool   `json:"PersonalFastest"`
		Team            string `json:"Team"`
		TeamColour      string `json:"TeamColour"`
	} `json:"Lines"`
}

type TimingStats struct {
	Withheld bool `json:"Withheld"`
	Lines    map[string]struct {
		Line                int    `json:"Line"`
		RacingNumber        string `json:"RacingNumber"`
		PersonalBestLapTime struct {
			Value    string `json:"Value"`
			Lap      int    `json:"Lap"`
			Position int    `json:"Position"`
		} `json:"PersonalBestLapTime"`
		BestSectors []struct {
			Value    string `json:"Value"`
			Position int    `json:"Position"`
		} `json:"BestSectors"`
		BestSpeeds struct {
			I1 struct {
				Value    string `json:"Value"`
				Position int    `json:"Position"`
			} `json:"I1"`
			I2 struct {
				Value    string `json:"Value"`
				Position int    `json:"Position"`
			} `json:"I2"`
			Fl struct {
				Value    string `json:"Value"`
				Position int    `json:"Position"`
			} `json:"FL"`
			St struct {
				Value    string `json:"Value"`
				Position int    `json:"Position"`
			} `json:"ST"`
		} `json:"BestSpeeds"`
	} `json:"Lines"`
}

type ReferenceTimingAppData struct {
	Lines map[string]struct {
		RacingNumber string  `json:"RacingNumber"`
		Line         int     `json:"Line"`
		GridPos      string  `json:"GridPos"`
		Stints       []Stint `json:"Stints"`
	} `json:"Lines"`
}

type ChangeTimingAppData struct {
	Lines map[string]DriverTimingAppData `json:"Lines"`
}

type DriverTimingAppData struct {
	RacingNumber string           `json:"RacingNumber"`
	Line         int              `json:"Line"`
	GridPos      string           `json:"GridPos"`
	Stints       map[string]Stint `json:"Stints"`
}

type Stint struct {
	LapFlags        int    `json:"LapFlags"`
	Compound        string `json:"Compound"`
	New             string `json:"New"`
	TyresNotChanged string `json:"TyresNotChanged"`
	TotalLaps       int    `json:"TotalLaps"`
	StartLaps       int    `json:"StartLaps"`
	LapTime         string `json:"LapTime"`
	LapNumber       int    `json:"LapNumber"`
}

type WeatherData struct {
	AirTemp       string `json:"AirTemp"`
	Humidity      string `json:"Humidity"`
	Pressure      string `json:"Pressure"`
	Rainfall      string `json:"Rainfall"`
	TrackTemp     string `json:"TrackTemp"`
	WindDirection string `json:"WindDirection"`
	WindSpeed     string `json:"WindSpeed"`
}

type TrackStatus struct {
	Status  string `json:"Status"`
	Message string `json:"Message"`
}

type DriverData struct {
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

type RaceControlMessagesMap struct {
	Messages map[string]RaceControlMessage `json:"Messages"`
}

type RaceControlMessages struct {
	Messages []RaceControlMessage `json:"Messages"`
}

type RaceControlMessage struct {
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

type SessionInfo struct {
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

type ReferenceSessionData struct {
	Series       []SessionDataSeries       `json:"Series"`
	StatusSeries []SessionDataStatusSeries `json:"StatusSeries"`
}

type ChangeSessionData struct {
	Series       map[string]SessionDataSeries       `json:"Series"`
	StatusSeries map[string]SessionDataStatusSeries `json:"StatusSeries"`
}

type SessionDataSeries struct {
	UTC time.Time `json:"Utc"`
	Lap uint8     `json:"Lap"`
}

type SessionDataStatusSeries struct {
	Utc           time.Time `json:"Utc"`
	TrackStatus   string    `json:"TrackStatus"`
	SessionStatus string    `json:"SessionStatus"`
}

type TimingData struct {
	Lines map[string]DriverTimingData `json:"Lines"`
}

type DriverTimingData struct {
	TimeDiffToFastest       string `json:"TimeDiffToFastest"`
	TimeDiffToPositionAhead string `json:"TimeDiffToPositionAhead"`
	Line                    int    `json:"Line"`
	Position                string `json:"Position"`
	ShowPosition            bool   `json:"ShowPosition"`
	RacingNumber            string `json:"RacingNumber"`
	Retired                 bool   `json:"Retired"`
	InPit                   bool   `json:"InPit"`
	PitOut                  bool   `json:"PitOut"`
	Stopped                 bool   `json:"Stopped"`
	Status                  int    `json:"Status"`
	GapToLeader             string `json:"GapToLeader"`
	IntervalToPositionAhead struct {
		Value    string `json:"Value"`
		Catching bool   `json:"Catching"`
	} `json:"IntervalToPositionAhead"`
	Sectors []SectorTiming `json:"Sectors"`
	Speeds  struct {
		I1 struct {
			Value           string `json:"Value"`
			Status          int    `json:"Status"`
			OverallFastest  bool   `json:"OverallFastest"`
			PersonalFastest bool   `json:"PersonalFastest"`
		} `json:"I1"`
		I2 struct {
			Value           string `json:"Value"`
			Status          int    `json:"Status"`
			OverallFastest  bool   `json:"OverallFastest"`
			PersonalFastest bool   `json:"PersonalFastest"`
		} `json:"I2"`
		Fl struct {
			Value           string `json:"Value"`
			Status          int    `json:"Status"`
			OverallFastest  bool   `json:"OverallFastest"`
			PersonalFastest bool   `json:"PersonalFastest"`
		} `json:"FL"`
		St struct {
			Value           string `json:"Value"`
			Status          int    `json:"Status"`
			OverallFastest  bool   `json:"OverallFastest"`
			PersonalFastest bool   `json:"PersonalFastest"`
		} `json:"ST"`
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

type SectorTiming struct {
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

type LapCount struct {
	CurrentLap uint8 `json:"CurrentLap"`
	TotalLaps  uint8 `json:"TotalLaps"`
}
