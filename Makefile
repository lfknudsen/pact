all:
	go build -C cmd/pact -o ../../pact.exe -buildvcs=false
	go build -C cmd/pact-server -o ../../pact-server.exe -buildvcs=false
	install pact.exe pact-server.exe
