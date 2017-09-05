BUILD_DATE := `date +%Y-%m-%d\ %H:%M`
VERSIONFILE := version.go
APP_VERSION := `bash ./generate_version.sh`
APP_NAME := "bungie-manifest-updater"

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
clean:
	rm $(APP_NAME)
