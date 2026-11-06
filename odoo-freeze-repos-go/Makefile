BIN := odoo-freeze-repos

.PHONY: build test vet fmt check install clean

build:  ## Compila el binario en bin/
	go build -o bin/$(BIN) ./cmd/$(BIN)

test:  ## Ejecuta todos los tests
	go test ./...

vet:  ## Analisis estatico
	go vet ./...

fmt:  ## Formatea el codigo
	gofmt -w .

check: fmt vet test  ## Formato + analisis + tests

install:  ## Instala el binario en ~/go/bin
	go install ./cmd/$(BIN)

clean:
	rm -rf bin
