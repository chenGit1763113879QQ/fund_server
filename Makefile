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

dailyinfo:
	python3 script/dailyinfo.py

database:
	python3 script/database.py

pip:
	pip3 install --upgrade -r requirements.txt -i https://pypi.douban.com/simple

compose:
	docker-compose up