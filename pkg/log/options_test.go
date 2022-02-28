package log

import (
	"fmt"
	"testing"

	"log"

	"github.com/stretchr/testify/assert"
)

func Test_Options_Validate(t *testing.T) {
	opts := &Options{
		Level:            "test",
		Format:           "test",
		EnableColor:      true,
		DisableCaller:    false,
		OutputPaths:      []string{"stdout", "./log"},
		ErrorOutputPaths: []string{"stderr"},
	}

	errs := opts.Validate()
	expected := `[unrecognized level: "test" not a valid log format: "test"]`
	assert.Equal(t, expected, fmt.Sprintf("%s", errs))
}

func Test_Options_New(t *testing.T) {
	opts := NewOptions()
	logger := New(opts)
	logger.Info("czw")
	log.Println("czw1")
}

func Test_Options_Build(t *testing.T) {
	opts := NewOptions()
	opts.Build()
	log.Println("czw1")
}
