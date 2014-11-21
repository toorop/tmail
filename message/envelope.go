package message

// envelope reprsente a message envelope
type Envelope struct {
	MailFrom string
	RcptTo   []string
}
