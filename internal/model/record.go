package model

import "time"

type Record struct {
	ID        string
	Data      []byte
	CreatedAt time.Time
}
