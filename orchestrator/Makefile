CWD:=$(shell pwd)
ARCH=amd64 arm64
OS:="linux" "darwin"

all: build
	
mk-outdir:
	@mkdir -p orchestrator

build: mk-outdir
	@for i in ${ARCH};do\
		for j in ${OS}; do\
			mkdir -p orchestrator/$$j/$$i;\
			echo "    building:" $$i/$$j;\
			GOOS=$$j GOARCH=$$i go build -v -o orchestrator/$$j/$$i/orchestrator;\
		done;\
	done;


clean:
	rm -rf orchestrator

.PHONY: clean