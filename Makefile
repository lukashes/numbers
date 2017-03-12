.PHONY: .build

test:
	go test -bench=.

build:
	@[ -d .build ] || mkdir .build
	CGO_ENABLED=0 go build -ldflags="-s -w" -o .build/numbers main.go
	file  .build/numbers
	du -h .build/numbers

docker:
	docker build -t numbers .

run: build docker
	docker run --net=host --rm numbers
