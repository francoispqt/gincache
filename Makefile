.PHONY: test
test: 
	go test -v

.PHONY: cover
cover: 
	go test -coverprofile=coverage.out

.PHONY: coverhtml
coverhtml: 
	go tool cover -html=coverage.out