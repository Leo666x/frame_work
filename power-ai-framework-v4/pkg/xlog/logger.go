package xlog

import (
	"fmt"
	"github.com/rs/zerolog"
	"os"
	"time"
)

var logger *zerolog.Logger

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	// 配置日志输出为文本格式
	// 创建一个 ConsoleWriter 实例
	writer := zerolog.ConsoleWriter{
		Out:        os.Stdout,                 // 设置输出目标为标准输出
		TimeFormat: "2006-01-02 15:04:05.000", // 设置时间格式
		NoColor:    true,                      // 禁用颜色输出

	}
	log := zerolog.New(writer).With().Timestamp().Logger()
	logger = &log
}

func LogInfoF(systemTrackCode, apiName, title, msg string) {
	logger.Info().Msg(fmt.Sprintf("[%s]-[%s]-[%s]-%s", systemTrackCode, apiName, title, msg))
}

func LogErrorF(systemTrackCode, apiName, title, msg string, err error) {
	if err != nil {
		logger.Error().Msg(fmt.Sprintf("[%s]-[%s]-[%s]-%s,err:%v", systemTrackCode, apiName, title, msg, err))
	} else {
		logger.Error().Msg(fmt.Sprintf("[%s]-[%s]-[%s]-%s", systemTrackCode, apiName, title, msg))
	}
}
