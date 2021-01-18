package main

import (
	"context"
	"fmt"
	"log"

	"github.com/openconfig/grpctunnel/cmd/server"
)

func main() {
	fmt.Println("starting")
	conf := server.Config{
		TunnelAddress: "localhost:4321",
		ListenAddress: "localhost:54323",
		CertFile:      "../localhost.crt",
		KeyFile:       "../localhost.key",
		Target:        "targetclient",
	}
	fmt.Println("running")
	if err := server.Run(context.Background(), conf); err == nil {
		fmt.Println(err.Error())
		log.Fatalf("Run() got success, want error")
	}
}
