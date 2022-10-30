CWD:=$(shell pwd)
ARCH=amd64 arm64
OS:="linux" "darwin"

all: build
	
mk-outdir:
	@mkdir -p simul

mk-simul-files:
	@go test -platform=cssl 2>&1 > /dev/null

build: mk-outdir mk-simul-files
	@for i in ${ARCH};do\
		for j in ${OS}; do\
			mkdir -p users/$$j/$$i;\
			echo "    building:" $$i/$$j;\
			GOOS=$$j GOARCH=$$i go build -v -o simul/$$j/$$i/simul;\
		done;\
	done;


clean:
	rm -rf users

.PHONY: clean