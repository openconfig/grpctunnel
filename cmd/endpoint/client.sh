#!/bin/bash
go run endpoint.go --tunnel_address=localhost:4321 \
--dial_address=localhost:54322 --target=target1 \
--cert_file_name=/usr/local/google/home/jxxu/Documents/projects/gnmi_dialout/test/github.com/openconfig/grpctunnel/example/localhost.crt