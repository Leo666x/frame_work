package xlog

import (
	"errors"
	"testing"
)

func TestLogInfo(t *testing.T) {
	LogInfoF("8974516465841", "agent-code1", "调用模型", "你好，废话真多啊")
}

func TestLogError(t *testing.T) {
	LogErrorF("8974516465841", "agent-code1", "调用模型", "你好，废话真多啊", errors.New("这是一个错误"))
}
