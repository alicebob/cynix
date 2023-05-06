build:
	go build

tidy:
	go mod tidy

ci:	
	echo hi there
	ls /home/
	ls -l /usr/bin/nix*
	make -C proj ci
