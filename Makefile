build:
	go build

tidy:
	go mod tidy

ci:	
	echo hi there
	nix-shell --run "make -C proj ci"
