package log

import (
	"testing"

	"go.uber.org/zap"
)

func TestLog(t *testing.T) {
	V(0).Info("info-test", zap.Any("key1", "value1"))
	V(5).Info("info-test1", zap.Any("key1", "value1"))
	Errorw("1234", "a", "1")
	Error("1234", zap.Any("key1", "value1"))
	logger := WithName("NewLog")
	logger.Info("abcd")
	logger = WithValues("key", "value")
	logger.Infow("hahaha", "k1", "1")
	logger.Info("hahaha", zap.Any("k1", "k2"))
	logger.Infof("hahaha%s-%s", "k1", "1")
	logger.Infow("hahaha", "k1", "1")
}
