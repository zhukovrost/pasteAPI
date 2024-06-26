package metrics

import (
	"database/sql"
	"expvar"
	"pasteAPI/internal/config"
	"runtime"
	"time"
)

func PostMetrics(dbStats sql.DBStats) {
	expvar.NewString("version").Set(config.Version)

	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))
	expvar.Publish("database", expvar.Func(func() interface{} {
		return dbStats
	}))
	expvar.Publish("timestamp", expvar.Func(func() interface{} {
		return time.Now().Unix()
	}))
}
