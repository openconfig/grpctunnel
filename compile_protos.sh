#!/bin/sh

# Copyright 2017 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

proto_imports=".:${GOPATH}/src"

# Go
protoc -I="${proto_imports}" \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    --go_out=. --go_opt=paths=source_relative proto/types/types.proto

protoc -I="${proto_imports}" \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    --go_opt=Mproto/types/types.proto=github.com/openconfig/grpctunnel/proto/types \
    --go_out=. --go_opt=paths=source_relative proto/tunnel/tunnel.proto

protoc -I="${proto_imports}" \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    --go_opt=Mproto/types/types.proto=github.com/openconfig/grpctunnel/proto/types \
    --go_out=. --go_opt=paths=source_relative cmd/target/proto/config/target_config.proto
