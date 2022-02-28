package log

import (
	"context"
	"fmt"
	"sync"

	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Defines common log fields.
const (
	KeyRequestID   string = "requestID"
	KeyUsername    string = "username"
	KeyWatcherName string = "watcher"
)

// InfoLogger 记录一般日志信息
type InfoLogger interface {
	Info(msg string, fields ...zapcore.Field)
	Infof(format string, v ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Enabled() bool
}

// Logger 记录各种信息
type Logger interface {
	InfoLogger

	Debug(msg string, fields ...zapcore.Field)
	Debugf(format string, v ...interface{})
	Debugw(msg string, keysAndValues ...interface{})
	Warn(msg string, fields ...zapcore.Field)
	Warnf(format string, v ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Error(msg string, fields ...zapcore.Field)
	Errorf(format string, v ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Panic(msg string, fields ...zapcore.Field)
	Panicf(format string, v ...interface{})
	Panicw(msg string, keysAndValues ...interface{})
	Fatal(msg string, fields ...zapcore.Field)
	Fatalf(format string, v ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})

	V(level int) InfoLogger
	Write(p []byte) (n int, err error)
	WithValues(keysAndValues ...interface{}) Logger
	WithName(name string) Logger

	// WithContext returns a copy of context in which the log value is set.
	// WithContext(ctx context.Context) context.Context

	// Flush calls the underlying Core's Sync method, flushing any buffered
	// log entries. Applications should take care to call Sync before exiting.
	Flush()
}

var (
	std = New(NewOptions())
	mu  sync.Mutex
)

var _ Logger = &zapLogger{}

// Init 通过传入的Options初始化Logger
func Init(opts *Options) {
	mu.Lock()
	defer mu.Unlock()

	std = New(opts)
}

type infoLogger struct {
	level zapcore.Level
	log   *zap.Logger
}

func (l *infoLogger) Enabled() bool { return true }

func (l *infoLogger) Info(msg string, fields ...zapcore.Field) {
	if checkedEntry := l.log.Check(l.level, msg); checkedEntry != nil {
		checkedEntry.Write(fields...)
	}
}

func (l *infoLogger) Infof(format string, args ...interface{}) {
	if checkedEntry := l.log.Check(l.level, fmt.Sprintf(format, args...)); checkedEntry != nil {
		checkedEntry.Write()
	}
}

func (l *infoLogger) Infow(msg string, keysAndValues ...interface{}) {
	if checkedEntry := l.log.Check(l.level, msg); checkedEntry != nil {
		checkedEntry.Write(handleFields(l.log, keysAndValues)...)
	}
}

type zapLogger struct {
	zapLogger *zap.Logger
	infoLogger
}

type noopInfoLogger struct{}

func (l *noopInfoLogger) Enabled() bool                     { return false }
func (l *noopInfoLogger) Info(_ string, _ ...zapcore.Field) {}
func (l *noopInfoLogger) Infof(_ string, _ ...interface{})  {}
func (l *noopInfoLogger) Infow(_ string, _ ...interface{})  {}

var disabledInfoLogger = &noopInfoLogger{}

func handleFields(l *zap.Logger, args []interface{}, additional ...zapcore.Field) []zapcore.Field {
	if len(args) == 0 {
		return additional
	}

	fields := make([]zapcore.Field, 0, len(args)/2+len(additional))

	for i := 0; i < len(args); {
		if _, ok := args[i].(zapcore.Field); ok {
			l.DPanic("strongly-typed Zap Field passed to logr", zap.Any("zap field", args[i]))

			break
		}

		if i == len(args)-1 {
			l.DPanic("odd number of arguments passed as key-value pairs for logging", zap.Any("ignored key", args[i]))

			break
		}

		key, val := args[i], args[i+1]
		keyStr, isString := key.(string)
		if !isString {
			// if the key isn't a string, DPanic and stop logging
			l.DPanic(
				"non-string key argument passed to logging, ignoring all later arguments",
				zap.Any("invalid key", key),
			)

			break
		}

		fields = append(fields, zap.Any(keyStr, val))
		i += 2
	}

	return append(fields, additional...)
}

func NewLogger(l *zap.Logger) Logger {
	return &zapLogger{
		zapLogger: l,
		infoLogger: infoLogger{
			log:   l,
			level: zap.InfoLevel,
		},
	}
}

func New(opts *Options) *zapLogger {
	if opts == nil {
		opts = NewOptions()
	}

	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(opts.Level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	encodeLevel := zapcore.CapitalLevelEncoder
	if opts.Format == consoleFormat && opts.EnableColor {
		encodeLevel = zapcore.CapitalColorLevelEncoder
	}

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "timestamp",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    encodeLevel,
		EncodeTime:     timeEncoder,
		EncodeDuration: milliSecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	loggerConfig := &zap.Config{
		Level:             zap.NewAtomicLevelAt(zapLevel),
		Development:       opts.Development,
		DisableCaller:     opts.DisableCaller,
		DisableStacktrace: opts.DisableStacktrace,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         opts.Format,
		EncoderConfig:    encoderConfig,
		OutputPaths:      opts.OutputPaths,
		ErrorOutputPaths: opts.ErrorOutputPaths,
	}

	l, err := loggerConfig.Build(zap.AddStacktrace(zapcore.PanicLevel), zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}

	logger := &zapLogger{
		zapLogger: l.Named(opts.Name),
		infoLogger: infoLogger{
			level: zap.InfoLevel,
			log:   l,
		},
	}

	zap.RedirectStdLog(l)
	return logger
}

func SugaredLogger() *zap.SugaredLogger {
	return std.zapLogger.Sugar()
}

func StdErrorLogger() *log.Logger {
	if std == nil {
		return nil
	}

	if l, err := zap.NewStdLogAt(std.zapLogger, zapcore.ErrorLevel); err == nil {
		return l
	}

	return nil
}

func StdInfoLogger() *log.Logger {
	if std == nil {
		return nil
	}

	if l, err := zap.NewStdLogAt(std.zapLogger, zapcore.InfoLevel); err == nil {
		return l
	}

	return nil
}

func V(level int) InfoLogger {
	return std.V(level)
}

func (l *zapLogger) V(level int) InfoLogger {
	lvl := zapcore.Level(level)
	if l.zapLogger.Core().Enabled(lvl) {
		return &infoLogger{
			level: lvl,
			log:   l.zapLogger,
		}
	}

	return disabledInfoLogger
}

func (l *zapLogger) Write(p []byte) (n int, err error) {
	l.zapLogger.Info(string(p))
	return len(p), nil
}

func WithValues(keysAndValues ...interface{}) Logger {
	return std.WithValues(keysAndValues...)
}

func (l *zapLogger) WithValues(keysAndValues ...interface{}) Logger {
	newLogger := l.zapLogger.With(handleFields(l.zapLogger, keysAndValues)...)
	return NewLogger(newLogger)
}

func WithName(name string) Logger {
	return std.WithName(name)
}

func (l *zapLogger) WithName(name string) Logger {
	newLogger := l.zapLogger.Named(name)
	return NewLogger(newLogger)
}

func (l *zapLogger) Flush() {
	l.zapLogger.Sync()
}

func Flush() {
	std.Flush()
}

func ZapLogger() *zap.Logger {
	return std.zapLogger
}

func (l *zapLogger) Debug(msg string, fields ...zapcore.Field) {
	l.zapLogger.Debug(msg, fields...)
}

func (l *zapLogger) Debugf(format string, v ...interface{}) {
	l.zapLogger.Sugar().Debugf(format, v...)
}

func (l *zapLogger) Debugw(msg string, v ...interface{}) {
	l.zapLogger.Sugar().Debugw(msg, v...)
}

func Debug(msg string, fields ...zapcore.Field) {
	std.Debug(msg, fields...)
}

func Debugf(format string, v ...interface{}) {
	std.Debugf(format, v...)
}

func Debugw(msg string, v ...interface{}) {
	std.Debugw(msg, v...)
}

func (l *zapLogger) Warn(msg string, fields ...zapcore.Field) {
	l.zapLogger.Warn(msg, fields...)
}

func (l *zapLogger) Warnf(format string, v ...interface{}) {
	l.zapLogger.Sugar().Warnf(format, v...)
}

func (l *zapLogger) Warnw(msg string, v ...interface{}) {
	l.zapLogger.Sugar().Warnw(msg, v...)
}

func Warn(msg string, fields ...zapcore.Field) {
	std.Warn(msg, fields...)
}

func Warnf(format string, v ...interface{}) {
	std.Warnf(format, v...)
}

func Warnw(msg string, v ...interface{}) {
	std.Warnw(msg, v...)
}

func (l *zapLogger) Error(msg string, fields ...zapcore.Field) {
	l.zapLogger.Error(msg, fields...)
}

func (l *zapLogger) Errorf(format string, v ...interface{}) {
	l.zapLogger.Sugar().Errorf(format, v...)
}

func (l *zapLogger) Errorw(msg string, v ...interface{}) {
	l.zapLogger.Sugar().Errorw(msg, v...)
}

func Error(msg string, fields ...zapcore.Field) {
	std.Error(msg, fields...)
}

func Errorf(format string, v ...interface{}) {
	std.Errorf(format, v...)
}

func Errorw(msg string, v ...interface{}) {
	std.Errorw(msg, v...)
}

func (l *zapLogger) Panic(msg string, fields ...zapcore.Field) {
	l.zapLogger.Panic(msg, fields...)
}

func (l *zapLogger) Panicf(format string, v ...interface{}) {
	l.zapLogger.Sugar().Panicf(format, v...)
}

func (l *zapLogger) Panicw(msg string, v ...interface{}) {
	l.zapLogger.Sugar().Panicw(msg, v...)
}

func Panic(msg string, fields ...zapcore.Field) {
	std.Panic(msg, fields...)
}

func Panicf(format string, v ...interface{}) {
	std.Panicf(format, v...)
}

func Panicw(msg string, v ...interface{}) {
	std.Panicw(msg, v...)
}

func (l *zapLogger) Fatal(msg string, fields ...zapcore.Field) {
	l.zapLogger.Fatal(msg, fields...)
}

func Fatal(msg string, fields ...zapcore.Field) {
	std.Fatal(msg, fields...)
}

func (l *zapLogger) Fatalf(format string, v ...interface{}) {
	l.zapLogger.Sugar().Fatalf(format, v...)
}

func Fatalf(format string, v ...interface{}) {
	std.Fatalf(format, v...)
}

func (l *zapLogger) Fatalw(msg string, v ...interface{}) {
	l.zapLogger.Sugar().Fatalw(msg, v...)
}

func Fatalw(msg string, v ...interface{}) {
	std.Fatalw(msg, v...)
}

func (l *zapLogger) Info(msg string, fields ...zapcore.Field) {
	l.zapLogger.Info(msg, fields...)
}

func Info(msg string, fields ...zapcore.Field) {
	std.Info(msg, fields...)
}

func (l *zapLogger) Infof(format string, v ...interface{}) {
	l.zapLogger.Sugar().Infof(format, v...)
}

func Infof(format string, v ...interface{}) {
	std.Infof(format, v...)
}

func (l *zapLogger) Infow(msg string, v ...interface{}) {
	l.zapLogger.Sugar().Infow(msg, v...)
}
func Infow(msg string, keysAndValues ...interface{}) {
	std.Infow(msg, keysAndValues...)
}

func L(ctx context.Context) *zapLogger {
	return std.L(ctx)
}

func (l *zapLogger) L(ctx context.Context) *zapLogger {
	ng := l.clone()

	if requestID := ctx.Value(KeyRequestID); requestID != nil {
		ng.zapLogger = ng.zapLogger.With(zap.Any(KeyRequestID, requestID))
	}

	if userName := ctx.Value(KeyUsername); userName != nil {
		ng.zapLogger = ng.zapLogger.With(zap.Any(KeyUsername, userName))
	}

	if wathcherName := ctx.Value(KeyWatcherName); wathcherName != nil {
		ng.zapLogger = ng.zapLogger.With(zap.Any(KeyWatcherName, wathcherName))
	}

	return ng
}

//nolint:predeclared
func (l *zapLogger) clone() *zapLogger {
	copy := *l

	return &copy
}
