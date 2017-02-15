package core

func init() {
	TmailPlugins = make(map[string][]TmailPlugin)
	SMTPdPlugins = make(map[string][]SMTPdPlugin)
	DeliverdPlugins = make(map[string][]DeliverdPlugin)
}

// Tmail core plugin

// TmailPlugin base plugin for hooks:
// - postinit
type TmailPlugin func()

// TmailPlugins is a map of plugin
var TmailPlugins map[string][]TmailPlugin

// RegisterPlugin registers a new plugin
func RegisterPlugin(hook string, plugin TmailPlugin) {
	TmailPlugins[hook] = append(TmailPlugins[hook], plugin)
}

func execTmailPlugins(hook string) {
	if plugins, found := TmailPlugins[hook]; found {
		for _, plugin := range plugins {
			plugin()
		}
	}
	return
}

// Smtpd plugins

// SMTPdPlugin is the type for SMTPd plugins
type SMTPdPlugin func(s *SMTPServerSession) bool

// SMTPdPlugins is a map of SMTPd plugins
var SMTPdPlugins map[string][]SMTPdPlugin

// RegisterSMTPdPlugin registers a new smtpd plugin
func RegisterSMTPdPlugin(hook string, plugin SMTPdPlugin) {
	SMTPdPlugins[hook] = append(SMTPdPlugins[hook], plugin)
}

func execSMTPdPlugins(hook string, s *SMTPServerSession) bool {
	if plugins, found := SMTPdPlugins[hook]; found {
		for _, plugin := range plugins {
			if plugin(s) {
				return true
			}
			return false
		}
	}
	return false
}

// Deliverd plugins

// DeliverdPlugin type for deliverd plugin
type DeliverdPlugin func(d *delivery)

// DeliverdPlugins map of deliverd plugins
var DeliverdPlugins map[string][]DeliverdPlugin

// RegisterDeliverdPlugin registers plugin for deliverd hooks
func RegisterDeliverdPlugin(hook string, plugin DeliverdPlugin) {
	DeliverdPlugins[hook] = append(DeliverdPlugins[hook], plugin)
}

func execDeliverdPlugins(hook string, d *delivery) {
	if plugins, found := DeliverdPlugins[hook]; found {
		for _, plugin := range plugins {
			plugin(d)
		}
	}
}
