build:
	go build -v ./cmd/gitopsish-server
	go build -v ./cmd/gitopsish

test:
	go test ./... -v

image:
	docker build -t igaskin/gitopsish