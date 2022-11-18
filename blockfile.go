package main

import (
	"time"
)

const (
	BlockFileSpecV1 = "https://rapidblock.org/spec/v1/"
)

type BlockFile struct {
	Spec        string           `json:"@spec"`
	PublishedAt time.Time        `json:"publishedAt"`
	Blocks      map[string]Block `json:"blocks"`
}

type Block struct {
	Reason        string    `json:"reason"`
	Tags          []string  `json:"tags"`
	DateRequested time.Time `json:"dateRequested"`
	DateDecided   time.Time `json:"dateDecided"`
}
