# JS에서 npm 명령어 등록하듯 Makefile로 명령어 등록

build:
	go build -o ./blockchain

run: build
	./blockchain

test:
	# ...은 현재 디렉터리와 하위 디렉터리 모두를 뜻함
	go test -v ./...
