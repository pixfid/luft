package data

import (
	"time"
)

type ActionType int8

const (
	Connected ActionType = iota
	Disconnected
	Unknown
)

type LogEvent struct {
	Date       time.Time
	ActionType ActionType
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
	FileName           string
	ExternalUsbIdsPath string
	SortBy             string
	Untrusted          bool
	Login              string
	Password           string
	Port               string
	IP                 string
	Number             int
}
