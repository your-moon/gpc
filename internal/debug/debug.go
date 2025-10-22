package debug

import (
	"fmt"
	"os"
	"strings"
)

var (
	DebugEnabled = false
	VerboseMode  = false
)

// SetDebugMode enables or disables debug output
func SetDebugMode(enabled bool) {
	DebugEnabled = enabled
}

// SetVerboseMode enables or disables verbose output
func SetVerboseMode(enabled bool) {
	VerboseMode = enabled
}

// Debug prints debug information if debug mode is enabled
func Debug(format string, args ...interface{}) {
	if DebugEnabled {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Verbose prints verbose information if verbose mode is enabled
func Verbose(format string, args ...interface{}) {
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "[VERBOSE] "+format+"\n", args...)
	}
}

// Info prints informational messages
func Info(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", args...)
}

// Warn prints warning messages
func Warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", args...)
}

// Error prints error messages
func Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}

// PassHeader prints a header for each analysis pass
func PassHeader(passName string) {
	if DebugEnabled {
		separator := strings.Repeat("=", 60)
		fmt.Fprintf(os.Stderr, "\n%s\n", separator)
		fmt.Fprintf(os.Stderr, "PASS: %s\n", passName)
		fmt.Fprintf(os.Stderr, "%s\n", separator)
	}
}

// PassFooter prints a footer for each analysis pass
func PassFooter(passName string, count int) {
	if DebugEnabled {
		fmt.Fprintf(os.Stderr, "PASS: %s completed - found %d items\n", passName, count)
		separator := strings.Repeat("-", 60)
		fmt.Fprintf(os.Stderr, "%s\n", separator)
	}
}

// Section prints a section header
func Section(title string) {
	if DebugEnabled {
		fmt.Fprintf(os.Stderr, "\n--- %s ---\n", title)
	}
}

// Item prints information about a single item
func Item(index int, format string, args ...interface{}) {
	if DebugEnabled {
		allArgs := make([]interface{}, 0, len(args)+1)
		allArgs = append(allArgs, index)
		allArgs = append(allArgs, args...)
		fmt.Fprintf(os.Stderr, "  [%d] "+format+"\n", allArgs...)
	}
}

// Indent prints indented information
func Indent(level int, format string, args ...interface{}) {
	if DebugEnabled {
		indent := strings.Repeat("  ", level)
		fmt.Fprintf(os.Stderr, indent+format+"\n", args...)
	}
}

// Stats prints statistics
func Stats(title string, stats map[string]interface{}) {
	if DebugEnabled {
		fmt.Fprintf(os.Stderr, "\n%s Statistics:\n", title)
		for key, value := range stats {
			fmt.Fprintf(os.Stderr, "  %s: %v\n", key, value)
		}
	}
}
