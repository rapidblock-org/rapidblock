package main

import (
	"time"
)

const BlockFileSpecV1 = "https://chronos-tachyon.net/fediblock/spec/v1/"

type BlockFile struct {
	Spec        string               `json:"@spec"`
	PublishedAt time.Time            `json:"publishedAt"`
	Blocks      map[string]BlockItem `json:"blocks"`
}

type BlockItem struct {
	DateReported  time.Time `json:"dateReported"`
	DateBlocked   time.Time `json:"dateBlocked"`
	IsRacism      bool      `json:"isRacism"`
	IsMisogyny    bool      `json:"isMisogyny"`
	IsQueerphobia bool      `json:"isQueerphobia"`
	IsHarassment  bool      `json:"isHarassment"`
	IsFraud       bool      `json:"isFraud"`
	Reason        string    `json:"reason"`
	ReceiptsURL   string    `json:"reciptsURL"`
}
