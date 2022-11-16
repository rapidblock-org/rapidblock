package main

import (
	"net/url"
	"time"
)

const BlockFileSpecV1 = "https://fediblock.online/spec/v1/"

type BlockFile struct {
	Spec        string           `json:"@spec"`
	PublishedAt time.Time        `json:"publishedAt"`
	Blocks      map[string]Block `json:"blocks"`
}

type Block struct {
	DateReported   time.Time `json:"dateReported"`
	DateBlocked    time.Time `json:"dateBlocked"`
	Reason         string    `json:"reason"`
	ReceiptsURL    string    `json:"reciptsURL"`
	IsRacism       bool      `json:"isRacism"`
	IsAntisemitism bool      `json:"isAntisemitism"`
	IsMisogyny     bool      `json:"isMisogyny"`
	IsQueerphobia  bool      `json:"isQueerphobia"`
	IsHarassment   bool      `json:"isHarassment"`
	IsFraud        bool      `json:"isFraud"`
	IsCopyright    bool      `json:"isCopyright"`
}

type PrivateBlock struct {
	Block
	Reporter *url.URL
	Receipts *url.URL
	Domain   string
	IsBlock  bool
}
