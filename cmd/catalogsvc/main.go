package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	webDir := flag.String("web", "web", "directory containing the browse page")
	flag.Parse()

	server := BuildServer(*webDir)
	done := make(chan error, 1)
	go func() {
		done <- server.Start(*addr)
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-done:
		if err != nil {
			log.Fatalf("catalog service stopped: %v", err)
		}
	case sig := <-signals:
		log.Printf("received %v, shutting down", sig)
		_ = server.Shutdown()
	}
}
