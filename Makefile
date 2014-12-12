gossiper: main.go
	go build .

clean:
	go clean

deps:
	go get -d

.PHONY: clean deps