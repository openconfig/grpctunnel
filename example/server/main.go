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
		ListenAddress: "localhost:54321",
		CertFile:      "/usr/local/google/home/jxxu/Documents/projects/gnmi_dialout/test/gnmi/testing/fake/gnmi/cmd/fake_server/fakecrt.crt",
		KeyFile:       "/usr/local/google/home/jxxu/Documents/projects/gnmi_dialout/test/gnmi/testing/fake/gnmi/cmd/fake_server/fakekey.key",
		// CertFile: "/usr/local/google/home/jxxu/Documents/projects/gnmi_dialout/test/gnmi/cmd/gnmi_collector/docker/config/cert.pem",
		// KeyFile:  "/usr/local/google/home/jxxu/Documents/projects/gnmi_dialout/test/gnmi/cmd/gnmi_collector/docker/config/key.pem",
		// CertFile: "/usr/local/google/home/jxxu/Documents/projects/gnmi_dialout/test/gnmi/cmd/gnmi_collector/testdata/good.crt",
		// KeyFile:  "/usr/local/google/home/jxxu/Documents/projects/gnmi_dialout/test/gnmi/cmd/gnmi_collector/testdata/good.key",
		Target: "target1",
	}
	fmt.Println("running")
	if err := server.Run(context.Background(), conf); err == nil {
		fmt.Println(err.Error())
		log.Fatalf("Run() got success, want error")
	}
}
