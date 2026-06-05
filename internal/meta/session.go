package meta

import "time"

type Session struct {
	UUID      string
	Name      string
	Type      string
	Status    string
	CreatedAt time.Time
}
