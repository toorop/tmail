package core

// SMTPdPlugin is the type for SMTPd plugins
type SMTPdPlugin func(s *SMTPServerSession) bool

// SMTPdPlugins is a map of SMTPd plugins
var SMTPdPlugins map[string][]SMTPdPlugin

func init() {
	SMTPdPlugins = make(map[string][]SMTPdPlugin)
}

// RegisterSMTPdPlugin registers a new plugin
// TODO check hook
func RegisterSMTPdPlugin(hook string, plugin SMTPdPlugin) {
	SMTPdPlugins[hook] = append(SMTPdPlugins[hook], plugin)
}
