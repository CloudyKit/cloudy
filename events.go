package cloudy

import "github.com/CloudyKit/cloudy/event"

type RunServerEvent struct {
	event.Event
	Host string
	Port string
}

type RunServerEventTLS struct {
	event.Event
	Host     string
	CertFile string
	KeyFile  string
}
