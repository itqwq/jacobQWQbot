package log

import "github.com/sirupsen/logrus"

// WarnLevel 表示警告日志级别。
var WarnLevel = logrus.WarnLevel

// InfoLevel 表示信息日志级别。
var InfoLevel = logrus.InfoLevel

// DebugLevel 表示调试日志级别。
var DebugLevel = logrus.DebugLevel

// ErrorLevel 表示错误日志级别。
var ErrorLevel = logrus.ErrorLevel

// FatalLevel 表示致命错误日志级别。
var FatalLevel = logrus.FatalLevel

// PanicLevel 表示恐慌日志级别。
var PanicLevel = logrus.PanicLevel

// TextFormatter 是 logrus 中的文本格式化器的别名。
type TextFormatter = logrus.TextFormatter

// Level 是 logrus 中级别的别名。
type Level = logrus.Level

// CheckErr 检查错误是否不为 nil，并将其记录在提供的日志级别上。
func CheckErr(level logrus.Level, err error) {
	if err != nil {
		Log(level, err)
	}
}

// Log 在指定的日志级别上记录提供的消息。这个消息可以是任意类型任意数量
func Log(level logrus.Level, messages ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		logrus.Info(messages...)
	case logrus.WarnLevel:
		logrus.Warn(messages...)
	case logrus.ErrorLevel:
		logrus.Error(messages...)
	case logrus.FatalLevel:
		logrus.Fatal(messages...)
	case logrus.PanicLevel:
		logrus.Panic(messages...)
	case logrus.DebugLevel:
		fallthrough
	default:
		logrus.Debug(messages...)
	}
}

// SetFormatter 设置 logrus 的格式化器。例如，您可以使用 &logrus.JSONFormatter{} 来设置为 JSON 格式输出日志
func SetFormatter(formatter logrus.Formatter) {
	logrus.SetFormatter(formatter)
}

// SetLevel 设置日志级别。您可以通过传入 logrus.Level 类型的值来设置不同的日志级别，例如 logrus.InfoLevel、logrus.WarnLevel、logrus.ErrorLevel 等。只有大于或等于所设置级别的日志消息才会被记录。
func SetLevel(level logrus.Level) {
	logrus.SetLevel(level)
}

// WithField 添加字段到日志记录。
func WithField(key string, value interface{}) *logrus.Entry {
	return logrus.WithField(key, value)
}

// WithFields 添加字段到日志记录。
func WithFields(fields logrus.Fields) *logrus.Entry {
	return logrus.WithFields(fields)
}

// Info 记录信息日志。
func Info(messages ...interface{}) {
	logrus.Info(messages...)
}

// Infof 格式化并记录信息日志。接受一个 字符串的参数 ，然后接收任意类型任意数量的数 插入到这个字符串中
func Infof(format string, messages ...interface{}) {
	logrus.Infof(format, messages...)
}

// Warn 记录警告日志。
func Warn(messages ...interface{}) {
	logrus.Warn(messages...)
}

// Warnf 格式化并记录警告日志。
func Warnf(format string, messages ...interface{}) {
	logrus.Warnf(format, messages...)
}

// Error 记录错误日志。
func Error(messages ...interface{}) {
	logrus.Error(messages...)
}

// Errorf 格式化并记录错误日志。
func Errorf(format string, messages ...interface{}) {
	logrus.Errorf(format, messages...)
}

// Fatal 记录致命错误日志。
func Fatal(messages ...interface{}) {
	logrus.Fatal(messages...)
}

// Debug 记录调试日志。
func Debug(messages ...interface{}) {
	logrus.Debug(messages...)
}

// Debugf 格式化并记录调试日志。
func Debugf(format string, messages ...interface{}) {
	logrus.Debugf(format, messages...)
}
