package log

import (
	"context"
)

var dl *Logger

func init() {
	dl = &Logger{}
}

type clk struct{}

func Ctx(ctx context.Context) *Logger {
	if l, ok := ctx.Value(clk{}).(*Logger); ok {
		return l
	}

	return dl
}

func (l Logger) WithContext(ctx context.Context) context.Context {
	if _, ok := ctx.Value(clk{}).(*Logger); !ok && l.emit == nil {
		return ctx
	}

	return context.WithValue(ctx, clk{}, &l)
}
