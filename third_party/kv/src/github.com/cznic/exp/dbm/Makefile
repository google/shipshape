# Copyright 2014 The dbm Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

.PHONY: all clean editor later mem nuke todo

grep=--include=*.go
testbin=dbm.test

all: editor
	go build
	go vet
	go install
	golint .
	make todo

clean:
	rm -f *~ cov cov.html test.db

editor:
	go fmt
	go test -i
	go test

later:
	@grep -n $(grep) LATER * || true
	@grep -n $(grep) MAYBE * || true

mem:
	go test -c
	./$(testbin) -test.bench . -test.memprofile mem.out -test.memprofilerate 1
	go tool pprof --lines --web --alloc_space $(testbin) mem.out

nuke: clean
	go clean -i

todo:
	@grep -nr $(grep) ^[[:space:]]*_[[:space:]]*=[[:space:]][[:alpha:]][[:alnum:]]* * || true
	@grep -nr $(grep) TODO * || true
	@grep -nr $(grep) BUG * || true
	@grep -nr $(grep) println * || true
