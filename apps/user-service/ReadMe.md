# User Service

Микросервис управления пользователями, организациями и ролями. Отвечает за аутентификацию, авторизацию, привязку Telegram и управление сотрудниками.

## Технологии

- **Go 1.22**
- **PostgreSQL** — основное хранилище
- **Redis** — кэш сессий, rate limiting
- **gRPC** — API для взаимодействия с другими сервисами
- **JWT** — аутентификация
- **bcrypt** — хеширование паролей

## Структура

```
user-service/
├── cmd/
│ ├── di_container.go                         # Создание DI контейнера
│ ├── garce_shut_down.go                      # Описание функций gracefull shutdown
│ ├── health_check.go                         # Описание функций health check
│ ├── start_server.go                         # Описание функций запуска сервера
│ └── main.go                                 # точка входа
├── internal/
│ ├── converter/                              # конвертеры (grpc <---> домен)
│ │ ├── user_converter.go                     # конвертер для пользователей
│ │ └── org_converter.go                      # конвертер для организации
│ ├── deps/                                   # работа с зависимостями
│ │ └── deps.go                               # описание DI контейнера
│ ├── domain/                                 # бизнес-сущности
│ │ ├── organization.go                       # структура Organization
│ │ ├── user.go                               # структура User
│ │ ├── common.go                             # общие удобные функции
│ │ └── errors.go                             # domain errors
│ ├── server/
│ │ ├── interseptors/                         # Интерсепторы
│ │ |     ├── logging.go                      # Интерсептор для логирования
│ │ |     └── recovery.go                     # Интерсептор для ловли паники
│ │ ├── handler/                              # Слой хэндлеров (удовлетворяют grpc интерфейсу)
│ │ |     ├── grpc_analytics_integration.go   # Хэндлер для аналитики
│ │ |     ├── grpc_organization.go            # Хэндлер для организации
│ │ |     ├── grpc_task_integration.go        # Хэндлер для задач
│ │ |     ├── grpc_telegram.go                # Хэндлер для телеграмма
│ │ |     ├── grpc_user_crud.go               # Хэндлер для пользователей
│ │ |     └── handler.go                      # Общее описание структуры хэндлера
│ │ ├── service/                              # Слой севиса (бизнесс-логика)
│ │ |     ├── analytics.go
│ │ |     ├── organization.go
│ │ |     ├── tasks.go
│ │ |     ├── telegram.go
│ │ |     ├── user.go
│ │ |     └── service.go                      # Общее описание структуры сервиса
│ │ ├── repository/                           # Слой репозитория (работа с БД)
│ │ |     ├── cache.go                        # Репозиторий кэша
│ │ |     ├── db_repo.go                      # Репозиторий БД
│ │ |     ├── interfaces.go                   # Описание интерфейсов для сервисного слоя
│ │ |     └── repository.go                   # Составной репозиторий (общий для сервиса)
│ │ └── grpc_server.go                        # gRPC сервер
│ └── config/
│         └── config.go                       # Описание струкктуры конфига сервиса
├── yml-configs/
│    └── grpcServerConfig.yml                 # Конфиг GRPC сервера
├── Dockerfile
├── go.mod
├── go.sum
└── .env                                      # адреса .yml конфигов, конфиги DB и Redis
```

## Модель данных

### Организация (Organization)

| Поле                | Тип       | Описание                          |
| ------------------- | --------- | --------------------------------- |
| `id`                | UUID      | Первичный ключ                    |
| `name`              | string    | Название организации              |
| `subscription_tier` | string    | Тариф: basic, premium, enterprise |
| `billing_email`     | string    | Email для счетов                  |
| `created_at`        | timestamp | Дата создания                     |

### Пользователь (User)

| Поле               | Тип       | Описание                   |
| ------------------ | --------- | -------------------------- |
| `id`               | UUID      | Первичный ключ             |
| `organization_id`  | UUID      | Внешний ключ к организации |
| `email`            | string    | Уникальный email           |
| `password_hash`    | string    | Хеш пароля (bcrypt)        |
| `full_name`        | string    | Полное имя                 |
| `role`             | string    | owner, manager, employee   |
| `telegram_chat_id` | int64     | Telegram ID (уникальный)   |
| `created_at`       | timestamp | Дата создания              |

### Роли и права

| Роль         | Права                                                                              |
| ------------ | ---------------------------------------------------------------------------------- |
| **owner**    | Всё: управление организацией, создание/удаление пользователей, просмотр всех задач |
| **manager**  | Создание задач, назначение исполнителей, просмотр задач организации                |
| **employee** | Только свои задачи, изменение статуса                                              |

## gRPC API

### Сервис

```protobuf
service UserService {
    // Интеграция с сервисом задач
    rpc ValidateUser(ValidateUserRequest) returns (ValidateUserResponse);
    rpc CheckUserExists(CheckUserExistsRequest) returns (CheckUserExistsResponse);
    rpc GetUserByID(GetUserByIDRequest) returns (GetUserResponse);
    rpc BatchGetUsers(BatchGetUsersRequest) returns (BatchGetUsersResponse);

    // Пользователи (CRUD)
    rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
    rpc GetUser(GetUserRequest) returns (GetUserResponse);
    rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
    rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
    rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);

    // Telegram привязка
    rpc LinkTelegram(LinkTelegramRequest) returns (LinkTelegramResponse);
    rpc GetUserByTelegram(GetUserByTelegramRequest) returns (GetUserResponse);
    rpc GetMyProfile(GetMyProfileRequest) returns (GetUserResponse);

    // Аналитика
    rpc GetAllUsers(GetAllUsersRequest) returns (GetAllUsersResponse);
    rpc GetUsersByRole(GetUsersByRoleRequest) returns (GetUsersByRoleResponse);
    rpc GetUserRole(GetUserRoleRequest returns (GetUserRoleResponse);

    // Организации
    rpc GetOrganization(GetOrganizationRequest) returns (GetOrganizationResponse);
    rpc SetupInitialOrganization(SetupInitialOrganizationRequest) returns (SetupInitialOrganizationResponse);
}
```

### Процесс регистрации и привязки Telegram

Этап 1: Создание организации и владельца (через SQL скрипт или API)
Этап 2: Привязка Telegram (через бота):
-------- 1. Пользователь пишет боту /start
-------- 2. Бот запрашивает email
-------- 3. Бот вызывает user-service.LinkTelegram(email, chatID)
-------- 4. User-service обновляет поле telegram_chat_id
Этап 3: Добавление сотрудников (через бота или API):
-------- 1. Руководитель через бота (/add_user) вводит email и имя
-------- 2. Бот вызывает user-service.CreateUser()
-------- 3. Создаётся пользователь с ролью employee, telegram_chat_id = null
-------- 4. Сотрудник самостоятельно привязывает Telegram через /start

### Конфигурация

Переменная ------------------------- Описание ------------------------- По умолчанию
DB_HOST --------------------------- PostgreSQL ----------------------- хост localhost
DB_PORT --------------------------- PostgreSQL -------------------------- порт 5432
DB_USER --------------------------- PostgreSQL ------------------- пользователь postgres
DB_PASSWORD ----------------------- PostgreSQL ----------------------- пароль postgres
DB_NAME --------------------------- PostgreSQL -------------------------- БД taskdb
REDIS_HOST -------------------------- Redis -------------------------- хост localhost
REDIS_PORT -------------------------- Redis ----------------------------- порт 6379
GRPC_PORT ---------------------------- gRPC ---------------------------- порт 50052
JWT_SECRET ---------------------- Секрет для JWT ---------------------- обязательный
JWT_EXPIRE_HOURS -------------- Время жизни токена ---------------------- (часы) 24
BCRYPT_COST ----------------------- Стоимость --------------------------- bcrypt 10

### Безопасность

Пароли — хранятся только в виде bcrypt-хеша

JWT — подписывается секретом, время жизни 24 часа

gRPC — в продакшене рекомендуется TLS

Rate limiting — через Redis

### Graceful Shutdown

Сервис корректно завершает работу:

Перестаёт принимать новые gRPC запросы

Завершает текущие запросы

Закрывает соединения с БД и Redis

Завершает процесс

### Health Check
