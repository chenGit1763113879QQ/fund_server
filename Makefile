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

# prevent exposure of the tunnel config
sakura_frp:
	docker run -d \
		--restart=always \
		--pull=always \
		--name=sakura_frp \
		natfrp/frpc \
		-f xxx:xxx