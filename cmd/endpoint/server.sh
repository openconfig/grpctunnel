#!/bin/bash
go run endpoint.go --tunnel_address=localhost:4321 --listen_address=localhost:54321 \
--target=target1 --cert_file_name=/usr/local/google/home/jxxu/Documents/projects/gnmi_dialout/test/github.com/openconfig/grpctunnel/example/localhost.crt \
--cert_key_file=/usr/local/google/home/jxxu/Documents/projects/gnmi_dialout/test/github.com/openconfig/grpctunnel/example/localhost.key 