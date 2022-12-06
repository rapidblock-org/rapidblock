package blockfile

import (
	"time"
)

const (
	SpecV1 = "https://rapidblock.org/spec/v1/"
)

type BlockFile struct {
	Spec        string           `json:"@spec"`
	PublishedAt time.Time        `json:"publishedAt"`
	Blocks      map[string]Block `json:"blocks"`
}

type Block struct {
	IsBlocked     bool      `json:"isBlocked"`
	Reason        string    `json:"reason"`
	Tags          []string  `json:"tags"`
	DateRequested time.Time `json:"dateRequested"`
	DateDecided   time.Time `json:"dateDecided"`
}
