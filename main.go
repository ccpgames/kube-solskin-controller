package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/celestialorb/solskin/exporter"
	"github.com/celestialorb/solskin/suppressor"
)

func main() {
	stopper := make(chan os.Signal)

	signal.Notify(stopper, syscall.SIGTERM)
	signal.Notify(stopper, syscall.SIGINT)

	go exporter.Start(stopper)
	go suppressor.Start(stopper)

	<-stopper
}
