package core

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
)

var (
	baseTimestamp time.Time
)

//init inits timestamp on start time to be support of miniTS
func init() {
	baseTimestamp = time.Now()
}

type FileFormatter struct {
	// Set to true to bypass checking for a TTY before outputting colors.
	ForceColors bool

	// Force disabling colors.
	DisableColors bool

	// Disable timestamp logging. useful when output is redirected to logging
	// system that already adds timestamps.
	DisableTimestamp bool

	// Enable logging the full timestamp when a TTY is attached instead of just
	// the time passed since beginning of execution.
	FullTimestamp bool

	// TimestampFormat to use for display when a full timestamp is printed
	TimestampFormat string

	// The fields are sorted by default for a consistent output. For applications
	// that log extremely frequently and don't use the JSON formatter this may not
	// be desired.
	DisableSorting bool
}

//Format formats the log entry
func (f *FileFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	var keys []string = make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}

	if !f.DisableSorting {
		sort.Strings(keys)
	}
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	prefixFieldClashes(entry.Data)

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = "Mon Jan 2 15:04:05 -0700 MST 2006"
	}

	levelText := strings.ToUpper(entry.Level.String())[0:4]

	if !f.FullTimestamp {
		fmt.Fprintf(b, "%s[%04d] %-44s ", levelText, miniTS(), entry.Message)
	} else {
		fmt.Fprintf(b, "%s[%s] %-44s ", levelText, entry.Time.Format(timestampFormat), entry.Message)
	}
	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(b, " %s=", k)
		f.appendValue(b, v)
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

//miniTS return the mini timestamp (ie seconds since logger start)
func miniTS() int {
	return int(time.Since(baseTimestamp) / time.Second)
}

//needsQuoting quotes string if needed
func needsQuoting(text string) bool {
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.') {
			return true
		}
	}
	return false
}

//appendValue add value to log line output
func (f *FileFormatter) appendValue(b *bytes.Buffer, value interface{}) {
	switch value := value.(type) {
	case string:
		if !needsQuoting(value) {
			b.WriteString(value)
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	case error:
		errmsg := value.Error()
		if !needsQuoting(errmsg) {
			b.WriteString(errmsg)
		} else {
			fmt.Fprintf(b, "%q", errmsg)
		}
	default:
		fmt.Fprint(b, value)
	}
}

//appendKeyValue add key=value to log line output
func (f *FileFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {

	b.WriteString(key)
	b.WriteByte('=')
	f.appendValue(b, value)
	b.WriteByte(' ')
}

//prefixFieldClashes avoid conflict for time / msg / level indexes
func prefixFieldClashes(data logrus.Fields) {
	if t, ok := data["time"]; ok {
		data["fields.time"] = t
	}

	if m, ok := data["msg"]; ok {
		data["fields.msg"] = m
	}

	if l, ok := data["level"]; ok {
		data["fields.level"] = l
	}
}

//printColored need for logrus.TextFormatter compatibility
func (f *FileFormatter) printColored(b *bytes.Buffer, entry *logrus.Entry, keys []string, timestampFormat string) {
}
