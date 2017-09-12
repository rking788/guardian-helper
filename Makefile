BUILD_DATE := `date +%Y-%m-%d\ %H:%M`
VERSIONFILE := version.go
APP_VERSION := `bash ./generate_version.sh`
APP_NAME := "guardian-helper"

genversion:
	rm -f $(VERSIONFILE)
	@echo "package main" > $(VERSIONFILE)
	@echo "const (" >> $(VERSIONFILE)
	@echo "  VERSION = \"$(APP_VERSION)\"" >> $(VERSIONFILE)
	@echo "  BUILD_DATE = \"$(BUILD_DATE)\"" >> $(VERSIONFILE)
	@echo ")" >> $(VERSIONFILE)
build: genversion
	go build
install: genversion
	go install
deploy: genversion
	GOOS=linux GOARCH=amd64 go build
	scp ./$(APP_NAME) do:
	rm $(APP_NAME)
clean:
	rm $(APP_NAME)
