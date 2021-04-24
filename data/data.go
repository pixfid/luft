package data

import "time"

type LogEvent struct {
	Date       time.Time
	ActionType string
	LogLine    string
}

type Event struct {
	ConnectedTime     time.Time
	Host              string
	Vid               string
	Pid               string
	ProductName       string
	ManufacturerName  string
	SerialNumber      string
	ConnectionPort    string
	DisconnectionTime time.Time
	Trusted           bool
	IsMassStorage     bool
}

type ParseParams struct {
	LogPath            string
	WlPath             string
	OnlyMass           bool
	CheckWl            bool
	Export             bool
	Format             string
	ExternalUsbIds     bool
	ExternalUsbIdsPath string
	SortBy             string
	Untrusted          bool
	Login              string
	Password           string
	Port               string
	Ip                 string
	Number             int
	WlData             []string
}
