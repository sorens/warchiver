package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

type Level int

const (
	LevelNone Level = iota
	LevelError
	LevelInfo
	LevelDebug
	LevelVerbose
)

type Logger struct {
	emit  func(level Level, tag, msg string)
	level Level
}

func New(w io.Writer, level Level) Logger {
	if w == nil {
		return Logger{}
	}
	return Logger{
		level: level,
		emit: func(_ Level, tag, msg string) {
			if tag != "" {
				fmt.Fprintf(w, "[%s] %s\n", tag, msg)
			} else {
				fmt.Fprintf(w, "%s\n", msg)
			}
		},
	}
}

// slogLevelVerbose is below slog.LevelDebug so walog's Verbose tier renders
// distinctly in JSON. walog already gates by its own Level; the slog handler
// is configured with a floor low enough not to double-filter.
const slogLevelVerbose = slog.Level(-8)

func NewJSON(w io.Writer, level Level) Logger {
	if w == nil {
		return Logger{}
	}
	sl := slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slogLevelVerbose}))
	return Logger{
		level: level,
		emit: func(lv Level, tag, msg string) {
			var slv slog.Level
			switch lv {
			case LevelVerbose:
				slv = slogLevelVerbose
			case LevelDebug:
				slv = slog.LevelDebug
			case LevelInfo:
				slv = slog.LevelInfo
			case LevelError:
				fallthrough
			default:
				slv = slog.LevelError
			}
			sl.LogAttrs(context.Background(), slv, msg)
		},
	}
}

func FromSlog(ctx context.Context, logger *slog.Logger) Logger {
	if logger == nil {
		return Logger{}
	}
	return Logger{
		level: LevelVerbose,
		emit: func(lv Level, tag, msg string) {
			var slv slog.Level
			switch lv {
			case LevelVerbose:
				slv = slogLevelVerbose
			case LevelDebug:
				slv = slog.LevelDebug
			case LevelInfo:
				slv = slog.LevelInfo
			default:
				slv = slog.LevelError
			}
			if tag != "" {
				logger.LogAttrs(ctx, slv, msg, slog.String("tag", tag))
				return
			}
			logger.Log(ctx, slv, msg)
		},
	}
}

func (l Logger) enabled(level Level) bool {
	return l.emit != nil && l.level >= level
}

func (l Logger) Info(format string, v ...any) {
	if l.enabled(LevelInfo) {
		l.emit(LevelInfo, "", fmt.Sprintf(format, v...))
	}
}

func (l Logger) Error(err error, format string, v ...any) {
	if l.emit == nil {
		return
	}
	msg := fmt.Sprintf(format, v...)
	if err != nil {
		if msg != "" {
			msg = fmt.Sprintf("%v: %s", err, msg)
		} else {
			msg = err.Error()
		}
	}
	l.emit(LevelError, "ERRR", msg)
}

func (l Logger) Debug(format string, v ...any) {
	if l.enabled(LevelDebug) {
		l.emit(LevelDebug, "DEBG", fmt.Sprintf(format, v...))
	}
}

func (l Logger) Verbose(format string, v ...any) {
	if l.enabled(LevelVerbose) {
		l.emit(LevelVerbose, "VERB", fmt.Sprintf(format, v...))
	}
}
