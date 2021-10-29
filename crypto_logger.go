package main

import log "github.com/sirupsen/logrus"

type cryptoLogger struct{}

func (f cryptoLogger) Error(message string, args ...interface{}) {
	log.Errorf(message, args...)
}

func (f cryptoLogger) Warn(message string, args ...interface{}) {
	log.Warnf(message, args...)
}

func (f cryptoLogger) Debug(message string, args ...interface{}) {
	log.Debugf(message, args...)
}

func (f cryptoLogger) Trace(message string, args ...interface{}) {
	log.Tracef(message, args...)
}
