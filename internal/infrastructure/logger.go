package infrastructure

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/franouer/proto-sync/internal/domain"
)

type ColorLogger struct {
	infoColor    *color.Color
	successColor *color.Color
	warningColor *color.Color
	errorColor   *color.Color
	debugColor   *color.Color
}

// NewColorLogger creates a new colorful logger
func NewColorLogger() domain.Logger {
	return &ColorLogger{
		infoColor:    color.New(color.FgBlue, color.Bold),
		successColor: color.New(color.FgGreen, color.Bold),
		warningColor: color.New(color.FgYellow, color.Bold),
		errorColor:   color.New(color.FgRed, color.Bold),
		debugColor:   color.New(color.FgMagenta),
	}
}

func (l *ColorLogger) Info(msg string, args ...interface{}) {
	prefix := l.infoColor.Sprint("[INFO]")
	fmt.Fprintf(os.Stderr, "%s %s\n", prefix, fmt.Sprintf(msg, args...))
}

func (l *ColorLogger) Success(msg string, args ...interface{}) {
	prefix := l.successColor.Sprint("[SUCCESS]")
	fmt.Fprintf(os.Stderr, "%s %s\n", prefix, fmt.Sprintf(msg, args...))
}

func (l *ColorLogger) Warning(msg string, args ...interface{}) {
	prefix := l.warningColor.Sprint("[WARNING]")
	fmt.Fprintf(os.Stderr, "%s %s\n", prefix, fmt.Sprintf(msg, args...))
}

func (l *ColorLogger) Error(msg string, args ...interface{}) {
	prefix := l.errorColor.Sprint("[ERROR]")
	fmt.Fprintf(os.Stderr, "%s %s\n", prefix, fmt.Sprintf(msg, args...))
}

func (l *ColorLogger) Debug(msg string, args ...interface{}) {
	prefix := l.debugColor.Sprint("[DEBUG]")
	fmt.Fprintf(os.Stderr, "%s %s\n", prefix, fmt.Sprintf(msg, args...))
}
