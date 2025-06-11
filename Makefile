.PHONY: build run test clean lint docker-build

APP_NAME=ws-gateway
BUILD_DIR=./build
MAIN_PATH=./cmd/ws-gateway

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

run:
	go run $(MAIN_PATH)

clean:
	rm -rf $(BUILD_DIR)

docker-build:
	docker build -t $(APP_NAME):latest .

docker-run:
	docker run --rm -p 8080:8080 $(APP_NAME):latest

help:
	@echo "Доступные команды:"
	@echo "  make build         - Компиляция приложения"
	@echo "  make run           - Запуск приложения"
	@echo "  make clean         - Удаление скомпилированных файлов"
	@echo "  make docker-build  - Сборка Docker-образа"
	@echo "  make docker-run    - Запуск Docker-контейнера" 