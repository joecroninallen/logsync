build:
	go build logsync.go
test:
	make -C filechunk test

clean:
	rm ./logsync
