package database

import "time"

type Car struct {
	Vid            string
	Sid            string
	Type           string
	Brand          string
	Model          string
	Seat           string
	Engine         string
	Shift          string
	ProductionYear string
	Description    string
	OrigPrice      int
	CurrPrice      int
	Contact        string
	Comments       []Comment
	UploadTime     string
	Hash           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Comment struct {
	Replier string
	Msg     string
}
