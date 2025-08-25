package main

import (
	"log"
)

func main() {
	srv, err := NewGRPCServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}
	log.Printf("server (TLS) listening at %v", srv.Addr())
	if err := srv.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
