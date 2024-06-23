package myLogger

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

var (
	once     sync.Once
	MyLogger *logrus.Logger
)

func New(needDebug bool) *logrus.Logger {
	once.Do(func() {
		MyLogger = logrus.New()
		MyLogger.SetFormatter(&CustomJSONFormatter{})
		if needDebug {
			MyLogger.SetLevel(logrus.DebugLevel)
		} else {
			MyLogger.SetLevel(logrus.InfoLevel)
		}
	})
	return MyLogger
}

type CustomJSONFormatter struct{}

func (f *CustomJSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(map[string]interface{})

	data["level"] = entry.Level.String()
	data["time"] = entry.Time.Format(time.RFC3339)
	data["message"] = entry.Message

	properties := make(map[string]interface{})
	for k, v := range entry.Data {
		properties[k] = v
	}

	if len(properties) > 0 {
		data["properties"] = properties
	}

	serialized, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %w", err)
	}
	return append(serialized, '\n'), nil
}
