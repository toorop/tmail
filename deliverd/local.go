package deliverd

// IsValidLocalRcpt checks if rcpt is a valid local destination
// 1 - catchall
// 2 - Mailbox (or wildcard)
// 3 - Alias
func IsValidLocalRcpt(rcpt string) (bool, error) {
	return MailboxExists(rcpt)
}
