# Переменные
DOCKER_COMPOSE = docker-compose
COMPOSE_USER_SERVICE_FILE = deployments/user-service/docker-compose.yml
USER_SERVICE_ENV_FILE = apps/user-service/.env
USER_SERVICE_COMPOSE_CMD = $(DOCKER_COMPOSE) -f $(COMPOSE_USER_SERVICE_FILE) --env-file $(USER_SERVICE_ENV_FILE)
COMPOSE_RABBITMQ_FILE = deployments/rabbitmq/docker-compose.yml
RABBITMQ_ENV_FILE = deployments/rabbitmq/.env
RABBITMQ_COMPOSE_CMD = $(DOCKER_COMPOSE) -f $(COMPOSE_RABBITMQ_FILE) --env-file $(RABBITMQ_ENV_FILE)
USER_SERVICE_CMD_PATH = apps/user-service/cmd

# Запуск всех контейнеров (user-service + rabbitmq)
.PHONY: up
up:
	@echo "[INFO] Starting all containers (user-service and RabbitMQ)..."
	@make --no-print-directory user_service_containers_up
	@make --no-print-directory rabbitmq_container_up
	@echo "[OK] All containers started"

# Остановка всех контейнеров (user-service + rabbitmq)
.PHONY: down
down:
	@echo "[INFO] Stopping all containers (user-service and RabbitMQ)..."
	@make --no-print-directory user_service_containers_down
	@make --no-print-directory rabbitmq_container_down
	@echo "[OK] All containers stopped"

# Статус всех контейнеров (user-service + rabbitmq)
.PHONY: status
status:
	@echo "[INFO] Status of all containers:"
	@echo ""
	@echo "=== User Service Containers ==="
	@make --no-print-directory user_service_containers_status
	@echo ""
	@echo "=== RabbitMQ Container ==="
	@make --no-print-directory rabbitmq_container_status

# Запуск всех контейнеров для user-service
.PHONY: user_service_containers_up
user_service_containers_up:
	@echo "[INFO] Starting all containers..."
	$(USER_SERVICE_COMPOSE_CMD) up -d
	@echo "[OK] Containers started"
	@make --no-print-directory user_service_containers_status

# Остановка всех контейнеров для user-service
.PHONY: user_service_containers_down
user_service_containers_down:
	@echo "[INFO] Stopping all containers..."
	$(USER_SERVICE_COMPOSE_CMD) down
	@echo "[OK] Containers stopped"

# Статус контейнеров для user-service
.PHONY: user_service_containers_status
user_service_containers_status:
	@echo "[INFO] Containers status:"
	$(USER_SERVICE_COMPOSE_CMD) ps

# Запуск контейнера для rabbitMQ
.PHONY: rabbitmq_container_up
rabbitmq_container_up:
	@echo "[INFO] Starting RabbitMQ container..."
	$(RABBITMQ_COMPOSE_CMD) up -d
	@echo "[OK] RabbitMQ Container started"
	@make --no-print-directory rabbitmq_container_status

# Остановка контейнера для rabbitMQ
.PHONY: rabbitmq_container_down
rabbitmq_container_down:
	@echo "[INFO] Stopping RabbitMQ container..."
	$(RABBITMQ_COMPOSE_CMD) down
	@echo "[OK] RAbbitMQ Container stopped"

# Статус контейнера для rabbitMQ
.PHONY: rabbitmq_container_status
rabbitmq_container_status:
	@echo "[INFO] Container status:"
	$(RABBITMQ_COMPOSE_CMD) ps

# Запуск user-service
.PHONY: user_service_start
user_service_start:
	echo "[INFO] Starting user-service"
	go run -C $(USER_SERVICE_CMD_PATH) .

# Показать все доступные команды
.PHONY: help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Main commands:"
	@echo "  make up												              - Start all containers (user-service + rabbitmq)"
	@echo "  make down												            - Stop all containers (user-service + rabbitmq)"
	@echo "  make status										              - Staus for all containers (user-service + rabbitmq)"
	@echo "  make user_service_containers_up              - Start all containers (for user-service)"
	@echo "  make user_service_containers_down            - Stop all containers (for user-service)"
	@echo "  make user_service_containers_status          - Status for all containers (for user-service)"
	@echo "  make rabbitmq_container_up                   - Start RabbitMQ container"
	@echo "  make rabbitmq_container_down                 - Stop RabbitMQ container"
	@echo "  make rabbitmq_container_status               - Status for RabbitMQ container"
	@echo "  make user_service_start                      - Start of user-service"












