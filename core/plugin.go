package core

// TmailPlugin base plugin for hooks:
// - postinit
type TmailPlugin func() error

// TmailPlugins is a map of plugin
var TmailPlugins map[string][]TmailPlugin

// SMTPdPlugin is the type for SMTPd plugins
type SMTPdPlugin func(s *SMTPServerSession) bool

// SMTPdPlugins is a map of SMTPd plugins
var SMTPdPlugins map[string][]SMTPdPlugin

func init() {
	TmailPlugins = make(map[string][]TmailPlugin)
	SMTPdPlugins = make(map[string][]SMTPdPlugin)
}

// RegisterPlugin registers a new plugin
func RegisterPlugin(hook string, plugin TmailPlugin) {
	TmailPlugins[hook] = append(TmailPlugins[hook], plugin)
}

// RegisterSMTPdPlugin registers a new smtpd plugin
// TODO check hook
func RegisterSMTPdPlugin(hook string, plugin SMTPdPlugin) {
	SMTPdPlugins[hook] = append(SMTPdPlugins[hook], plugin)
}

func execTmailPlugins(hook string) {
	if plugins, found := TmailPlugins[hook]; found {
		for _, plugin := range plugins {
			plugin()
		}
	}
	return
}
