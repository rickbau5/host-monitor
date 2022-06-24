package hostmonitor

import (
	"time"

	"github.com/go-logr/logr"
)

type HostMapOption interface {
	apply(*HostMap)
}

type optionFunc func(*HostMap)

func (f optionFunc) apply(hostMap *HostMap) {
	f(hostMap)
}

// HostOfflineTimeoutOption configures the time that must elapse before a host is considered inactive
func HostOfflineTimeoutOption(dur time.Duration) HostMapOption {
	return optionFunc(func(hostMap *HostMap) {
		hostMap.offlineTimeout = dur
	})
}

// LoggerOption configures the logger to be used for reporting non-critical errors
func LoggerOption(logger logr.Logger) HostMapOption {
	return optionFunc(func(hostMap *HostMap) {
		hostMap.logger = logger
	})
}
