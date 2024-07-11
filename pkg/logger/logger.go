package logger

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

func New(needDebug bool) *logrus.Logger {
	myLogger := logrus.New()
	myLogger.SetFormatter(&customJSONFormatter{})
	if needDebug {
		myLogger.SetLevel(logrus.DebugLevel)
	} else {
		myLogger.SetLevel(logrus.InfoLevel)
	}
	return myLogger
}

type customJSONFormatter struct{}

func (f *customJSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
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
