BUILD_DATE := `date +%Y-%m-%d\ %H:%M`
VERSIONFILE := version.go
APP_VERSION := `bash ./generate_version.sh`
APP_NAME := "guardian-helper"


all: build
genversion:
	rm -f $(VERSIONFILE)
	@echo "package main" > $(VERSIONFILE)
	@echo "const (" >> $(VERSIONFILE)
	@echo "  Version = \"$(APP_VERSION)\"" >> $(VERSIONFILE)
	@echo "  BuildDate = \"$(BUILD_DATE)\"" >> $(VERSIONFILE)
	@echo ")" >> $(VERSIONFILE)
build: genversion
	go build
install: genversion
	go install
test:
	go test -v ./...
coverage:
	## Right now the coverprofile option is not allowed when testing multiple packages.
	## this is the best we can do for now until writing a bash script to loop over packages.
	go test -cover ./...
#	go test --coverprofile=coverage.out
#	go tool cover -html=coverage.out
deploy: genversion
	GOOS=linux GOARCH=amd64 go build
	scp ./$(APP_NAME) do:
	rm $(APP_NAME)
clean:
	rm $(APP_NAME)
