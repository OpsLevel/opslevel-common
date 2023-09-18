package opslevel_common

import (
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"syscall"
)

var onlyOneSignalHandler = make(chan struct{})

// InitSignalHandler
// Usage:
//
//	func Start() {
//    log.Info().Msg("Starting...")
//    <-opslevel_common.InitSignalHandler() // Block until signals
//    log.Info().Msg("Stopping...")
//	}
func InitSignalHandler() <-chan struct{} {
	close(onlyOneSignalHandler) // panics when called twice

	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}

func Run(name string) {
	log.Info().Msgf("Starting %s ...", name)
	<-InitSignalHandler() // Block until signals
	log.Info().Msgf("Stopping %s ...", name)
}
