package main

import (
	"log"
	"os"
	"strconv"

	"github.com/tiagoposse/kscp-webhook/internal/server"
)

func main() {
	port := 443
	var err error
	if val, ok := os.LookupEnv("SERVER_PORT"); ok {
		port, err = strconv.Atoi(val)
		if err != nil {
			panic(err)
		}
	}

	s := server.NewServer(
		server.WithCertPath(os.Getenv("CERT_FILE_PATH")),
		server.WithKeyPath(os.Getenv("CERT_KEY_PATH")),
		server.WithPort(port),
	)

	if err := s.Serve(); err != nil {
		log.Fatalf("%v", err)
	}
}
