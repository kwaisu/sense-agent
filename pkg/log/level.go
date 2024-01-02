package log

// A Level is a logging priority. Higher levels are more important.
type Level int

/*
0 (emerg) corresponds to emerg in syslog, indicating system crash or unavailability
1 (alert) corresponds to an alert in syslog, indicating a situation that requires immediate action
2 (crit) corresponds to crit in syslog, indicating severe conditions such as hardware errors
3 (err) corresponds to err in syslog, indicating an error condition, less severe than errors at the urgent, alert, and critical levels
4 (warning) corresponds to the warning in syslog, indicating the conditions that need to be monitored
5 (notice) corresponds to notice in syslog, indicating conditions that are not errors but may require special handling
6 (info) corresponds to info in syslog, indicating an event or a non-error condition
7 (debug) corresponds to debug in syslog, indicating debugging or trace information
*/
const (
	LevelCritical Level = iota - 1
	// LevelError is logger error level.
	LevelError
	// LevelWarn is logger warn level.
	LevelWarning
	// LevelInfo is logger info level.
	LevelInfo
	// LevelDebug is logger debug level.
	LevelDebug
)

var priority2Levels = map[string]Level{
	"0": LevelCritical,
	"1": LevelCritical,
	"2": LevelCritical,
	"3": LevelError,
	"4": LevelWarning,
	"5": LevelInfo,
	"6": LevelInfo,
	"7": LevelDebug,
}

func String2Level(level string) Level {
	switch level {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN":
		return LevelWarning
	case "ERROR":
		return LevelError
	default:
		return -1
	}
}

func (l Level) string() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarning:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return ""
	}
}
