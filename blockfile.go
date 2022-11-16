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
	ReceiptsURL    string    `json:"reciptsURL,omitempty"`
	IsRacism       bool      `json:"isRacism,omitempty"`
	IsAntisemitism bool      `json:"isAntisemitism,omitempty"`
	IsMisogyny     bool      `json:"isMisogyny,omitempty"`
	IsQueerphobia  bool      `json:"isQueerphobia,omitempty"`
	IsHarassment   bool      `json:"isHarassment,omitempty"`
	IsFraud        bool      `json:"isFraud,omitempty"`
	IsCopyright    bool      `json:"isCopyright,omitempty"`
	IsSpam         bool      `json:"isSpam,omitempty"`
	IsMalware      bool      `json:"isMalware,omitempty"`
	IsCSAM         bool      `json:"isCSAM,omitempty"`
}

type PrivateBlock struct {
	Block
	Reporter *url.URL
	Receipts *url.URL
	Domain   string
	IsBlock  bool
}
