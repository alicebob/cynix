build:
	go build

tidy:
	go mod tidy

ci:	
	echo hi there
	make -C proj ci
