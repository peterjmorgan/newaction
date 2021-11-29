BINARY_NAME=newaction
BUILD_NUMBER_FILE=build-number.txt
OBJECTS=$(BINARY_NAME)

build: build-linux build-macos

build-linux: $(OBJECTS) $(BUILD_NUMBER_FILE)
	GOARCH=amd64 GOOS=linux go build -o ${BINARY_NAME}
	shasum ${BINARY_NAME}

build-macos:
	GOARCH=arm64 GOOS=darwin go build -o ${BINARY_NAME}-darwin_arm64

build_release: build $(OBJECTS) $(BUILD_NUMBER_FILE)
	# gh release create
	gh release create v0.$(BUILD_NUMBER) -n "v0.$(BUILD_NUMBER)" -t v0.$(BUILD_NUMBER) $(BINARY_NAME)
	gh release list

clean:
	go clean
	rm ${BINARY_NAME}
	rm ${BINARY_NAME}-darwin_arm64

include buildnumber.mak
