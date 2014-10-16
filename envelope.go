package main

// envelope reprsente a message envelope
type envelope struct {
	mailFrom string
	rcptTo   []string
}
