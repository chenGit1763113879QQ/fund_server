run:
	go run -tags=sonic,nomsgpack main.go

build:
	go build -tags=sonic,nomsgpack .

tidy:
	go env -w GOPROXY=https://goproxy.cn
	go get -u
	go mod tidy
	go clean

database:
	python3 script/database.py

dailyinfo:
	python3 script/dailyinfo.py

pip:
	pip3 install --upgrade -r requirements.txt -i https://pypi.douban.com/simple

compose:
	docker-compose up