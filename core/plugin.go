package core

// SMTPdPlugin is the type for SMTPd plugins
type SMTPdPlugin func(s *SMTPServerSession)

var newClientPlugin SMTPdPlugin

func init() {
	newClientPlugin = nil
}

// RegisterSMTPdPlugin registers a new plugin
func RegisterSMTPdPlugin(plugin SMTPdPlugin) {
	newClientPlugin = plugin
}
