//
// Copyright 2019 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package client

import (
	"context"
	"io"
	"testing"

	"github.com/openconfig/grpctunnel/tunnel"
)

var (
	testRH = func(u string) error {
		return nil
	}
	testH = func(_ string, _ io.ReadWriteCloser) error {
		return nil
	}
)

func TestRun(t *testing.T) {
	for _, test := range []struct {
		name            string
		tunnelAddress   string
		certFile        string
		target          string
		registerHandler tunnel.ClientRegHandlerFunc
		handler         tunnel.ClientHandlerFunc
	}{
		{
			name:     "InvalidCertFile",
			certFile: "fileDoesNotExist/NoFileHere",
		},
		{
			name:          "ClientRunFailure",
			tunnelAddress: "someDialAddress:20",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			conf := Config{
				TunnelAddress: test.tunnelAddress,
				CertFile:      test.certFile,
				Target:        test.target,
			}
			if err := Run(context.Background(), conf); err == nil {
				t.Fatal("Run() got success, want error")
			}
		})
	}
}
