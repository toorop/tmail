package message

// Envelope reprsente a message envelope
type Envelope struct {
	MailFrom string
	RcptTo   []string
}

func (e Envelope) String() string {
	out := "F" + e.MailFrom + "\000"
	for _, rcpt := range e.RcptTo {
		out += "T" + rcpt + "\000"
	}
	return out + "\000"
}
