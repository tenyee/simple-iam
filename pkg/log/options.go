package log

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// 日志打印格式类型
	consoleFormat = "console"
	jsonFormat    = "json"
)

// Options contains configuration items related to log.
type Options struct {
	OutputPaths       []string `json:"output-paths"`
	ErrorOutputPaths  []string `json:"error-output-paths"`
	Level             string   `json:"level"`
	Format            string   `json:"format"`
	DisableCaller     bool     `json:"disable-caller"`
	DisableStacktrace bool     `json:"disable-stacktrace"`
	EnableColor       bool     `json:"enable-color"`
	Development       bool     `json:"development"`
	Name              string   `json:"name"`
}

// NewOptions create a Options object with default parameters.
func NewOptions() *Options {
	return &Options{
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		Level:             zapcore.InfoLevel.String(),
		Format:            consoleFormat,
		DisableCaller:     false,
		DisableStacktrace: false,
		EnableColor:       false,
		Development:       false,
	}
}

// Validate valiate the options fields.
func (o *Options) Validate() []error {
	var errs []error

	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(o.Level)); err != nil {
		errs = append(errs, err)
	}

	format := strings.ToLower(o.Format)
	if format != consoleFormat && format != jsonFormat {
		errs = append(errs, fmt.Errorf("not a valid log format: %q", o.Format))
	}

	return errs
}

func (o *Options) String() string {
	data, _ := json.Marshal(o)

	return string(data)
}

// Build constructs a global zap logger from the Config and Options.
func (o *Options) Build() error {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(o.Level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	encodeLevel := zapcore.CapitalLevelEncoder
	if o.Format == consoleFormat && o.EnableColor {
		encodeLevel = zapcore.CapitalColorLevelEncoder
	}

	zapConfig := &zap.Config{
		Level:             zap.NewAtomicLevelAt(zapcore.Level(zapLevel)),
		Development:       o.Development,
		DisableCaller:     o.Development,
		DisableStacktrace: o.DisableStacktrace,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: o.Format,
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:     "message",
			LevelKey:       "level",
			NameKey:        "logger",
			StacktraceKey:  "stacktrace",
			CallerKey:      "caller",
			TimeKey:        "timestamp",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    encodeLevel,
			EncodeTime:     timeEncoder,
			EncodeDuration: milliSecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeName:     zapcore.FullNameEncoder,
		},
		OutputPaths:      o.OutputPaths,
		ErrorOutputPaths: o.ErrorOutputPaths,
	}

	logger, err := zapConfig.Build(zap.AddStacktrace(zapcore.PanicLevel))
	if err != nil {
		return err
	}

	zap.RedirectStdLog(logger.Named((o.Name)))
	zap.ReplaceGlobals(logger)
	return nil
}
