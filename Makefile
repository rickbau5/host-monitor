NAME=host-monitor
TAR_NAME=${NAME}.tar.gz
BINARY_DIR=bin
BINARY=${BINARY_DIR}/${NAME}
TAR=${BINARY_DIR}/${TAR_NAME}

# this depends on how the Raspberry Pi is set up
TARGET_ARGS=GOOS=linux GOARCH=arm CGO_ENABLED=0

package: build ${TAR}

.PHONY:
build: ${BINARY}
	echo ${BINARY}
	ls -l ${BINARY}

${BINARY}: vendor
	${TARGET_ARGS} go build -mod=vendor -o ${BINARY} .

.PHONY: vendor
vendor: vendor/vendor.txt
vendor/vendor.txt:
	go mod vendor

${TAR}: ${BINARY}
	pushd ${BINARY_DIR} && tar cvf ${TAR_NAME} ${NAME}; popd

.PHONY: clean
clean:
	go mod tidy
	@rm -rf ${BINARY_DIR}