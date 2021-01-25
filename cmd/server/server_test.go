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

package server

import (
	"context"
	"testing"

	"google3/third_party/golang/grpctunnel/tunnel/tunnel"
)

func TestListen(t *testing.T) {
	for _, test := range []struct {
		name, lAddr string
	}{
		{
			name:  "invalidListenAddress",
			lAddr: "invalid:999999999",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := listen(context.Background(), nil, test.lAddr, &[]tunnel.Target{}); err == nil {
				t.Fatalf("listen() got success, want error")
			}
		})
	}
}

func TestRun(t *testing.T) {
	for _, test := range []struct {
		name, lAddr, tAddr, certFile, keyFile string
	}{
		{
			name:     "InvalidCertFile",
			certFile: "does/Not\\Exist//",
			keyFile:  "does/Not\\Exist//",
		},
		{
			name:  "invalidTunnelAddress",
			lAddr: "invalid:999999999",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			conf := Config{
				TunnelAddress: test.tAddr,
				ListenAddress: test.lAddr,
				CertFile:      test.certFile,
				KeyFile:       test.keyFile,
			}
			if err := Run(context.Background(), conf); err == nil {
				t.Fatalf("Run() got success, want error")
			}
		})
	}
}
