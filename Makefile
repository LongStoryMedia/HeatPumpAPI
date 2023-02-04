SRC = $(CURDIR)

.SILENT:

init: 
	sh scripts/init.sh

build:
	GOOS=linux GOARCH=arm GOARM=7 go build -o prod/api $(SRC)

build-local:
	go build -o dev/api $(SRC)

start: build-local
	gin --bin=dev/api -p 3001 -a 8080 -i

kill:
	-PIDS="$(shell /usr/sbin/lsof -t -i:8080)"; kill -9 $$PIDS

.PHONY: build