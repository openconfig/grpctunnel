package main

import (
	"context"
	"log"

	"github.com/openconfig/grpctunnel/cmd/client"
)

func main() {

	conf := client.Config{
		TunnelAddress: "localhost:4321",  // ListenAddress of the server
		DialAddress:   "localhost:55321", // address to dial back to client
		CertFile:      "../localhost.crt",
		Target:        "target1",
	}
	if err := client.Run(context.Background(), conf); err == nil {
		log.Fatalf("Run() got success, want error: %v", err)
	}
}
