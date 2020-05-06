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

package bidi

import (
	"bytes"
	"errors"
	"sync"
	"testing"
)

var errRead = errors.New("read error")
var errWrite = errors.New("write error")

// rwc implements an io.ReadWriteCloser. Reads will read from the readBuf and
// writes will write to the writeBuf. Setting writeErr will cause writes to fail,
// and setting readErr will cause reads to fail. The waitgroup is used to ensure
// the successful goroutine does not return before the erroring goroutine.
// When an error is returned wg is decremented by one.
type rwc struct {
	writeErr, readErr bool
	readBuf           *bytes.Buffer
	writeBuf          *bytes.Buffer
	wg                *sync.WaitGroup
}

func (r *rwc) Write(p []byte) (int, error) {
	if r.writeErr {
		defer r.wg.Done()
		return 0, errWrite
	}
	r.wg.Wait()
	return r.writeBuf.Write(p)
}

func (r *rwc) Read(p []byte) (int, error) {
	if r.readErr {
		defer r.wg.Done()
		return 0, errRead
	}
	n, err := r.readBuf.Read(p)
	if err != nil || r.readErr {
		r.wg.Wait()
	}
	return n, err
}

func (r *rwc) Close() error {
	return nil
}

func newRWC(readErr, writeErr bool) *rwc {
	return &rwc{
		readBuf:  new(bytes.Buffer),
		writeBuf: new(bytes.Buffer),
		readErr:  readErr,
		writeErr: writeErr,
	}
}

func TestReadWrite(t *testing.T) {
	buf1 := "This arbitrary text!"
	buf2 := "Totally different text!"

	for _, test := range []struct {
		name       string
		buf1, buf2 *rwc
		err        error
		count      int
	}{
		{
			name: "fail_read",
			buf1: newRWC(true, false),
			buf2: &rwc{
				readBuf:  bytes.NewBufferString(buf1),
				writeBuf: new(bytes.Buffer),
			},
			err:   errRead,
			count: 1,
		},
		{
			name: "fail_read_reverse",
			buf1: &rwc{
				readBuf:  bytes.NewBufferString(buf2),
				writeBuf: new(bytes.Buffer),
			},
			buf2:  newRWC(true, false),
			err:   errRead,
			count: 1,
		},
		{
			name:  "failed_readers",
			buf1:  newRWC(true, false),
			buf2:  newRWC(true, false),
			err:   errRead,
			count: 2,
		},
		{
			name: "fail_write",
			buf1: newRWC(false, true),
			buf2: &rwc{
				readBuf:  bytes.NewBufferString(buf1),
				writeBuf: new(bytes.Buffer),
			},
			err:   errWrite,
			count: 1,
		},
		{
			name: "fail_write_reverse",
			buf1: &rwc{
				readBuf:  bytes.NewBufferString(buf2),
				writeBuf: new(bytes.Buffer),
			},
			buf2:  newRWC(false, true),
			err:   errWrite,
			count: 1,
		},
		{
			name: "fail_writers",
			buf1: &rwc{
				readBuf:  bytes.NewBufferString(buf2),
				writeBuf: new(bytes.Buffer),
				writeErr: true,
			},
			buf2: &rwc{
				readBuf:  bytes.NewBufferString(buf1),
				writeBuf: new(bytes.Buffer),
				writeErr: true,
			},
			err:   errWrite,
			count: 2,
		},
		{
			name: "success",
			buf1: &rwc{
				readBuf:  bytes.NewBufferString(buf2),
				writeBuf: new(bytes.Buffer),
			},
			buf2: &rwc{
				readBuf:  bytes.NewBufferString(buf1),
				writeBuf: new(bytes.Buffer),
			},
			err:   nil,
			count: 0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			wg := &sync.WaitGroup{}
			test.buf1.wg, test.buf2.wg = wg, wg
			wg.Add(test.count)
			err := Copy(test.buf1, test.buf2)
			if err != test.err {
				t.Fatalf("Copy() got %v, want %s.", err, test.err.Error())
			}
		})
	}
}
