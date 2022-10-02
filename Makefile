run:
	go run -tags=sonic,nomsgpack main.go

build:
	go build -tags=sonic,nomsgpack .

build-arm:
	GOOS=linux GOARCH=arm64 go build -tags=nomsgpack .

tidy:
	go env -w GOPROXY=https://goproxy.cn,direct
	go get -u
	go mod tidy
	go clean

compose:
	docker-compose up