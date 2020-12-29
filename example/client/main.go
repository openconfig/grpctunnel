package main

import (
	"context"
	"log"

	"github.com/openconfig/grpctunnel/cmd/client"
)

func main() {

	conf := client.Config{
		TunnelAddress: "localhost:54321", // ListenAddress of the server
		DialAddress:   "localhost:434",   // address to dial back to client
		CertFile:      "/usr/local/google/home/jxxu/Documents/projects/gnmi_dialout/test/gnmi/testing/fake/gnmi/cmd/fake_server/fakecrt.crt",
		// CertFile: "/usr/local/google/home/jxxu/Documents/projects/gnmi_dialout/test/gnmi/cmd/gnmi_collector/docker/config/cert.pem",
		// CertFile: "/usr/local/google/home/jxxu/Documents/projects/gnmi_dialout/test/gnmi/cmd/gnmi_collector/testdata/good.crt",
		Target: "target1",
	}
	if err := client.Run(context.Background(), conf); err == nil {
		log.Fatalf("Run() got success, want error: %v", err)
	}
}
