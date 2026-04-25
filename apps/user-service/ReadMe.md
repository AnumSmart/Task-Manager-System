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
│ ├── grace_shut_down.go                      # Описание функций gracefull shutdown
│ ├── health_check.go                         # Описание функций health check
│ ├── start_server.go                         # Описание функций запуска сервера
│ └── main.go                                 # точка входа
├── migrations/                               # миграции для user-service
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
│ │ ├── interceptors/                         # Интерсепторы
│ │ |     ├── jwt_auth.go                     # Интерсептор для авторизации (jwt)
│ │ |     ├── method_classifier.go            # Классификатор методов для использования интерсептором
│ │ |     ├── logging.go                      # Интерсептор для логирования
│ │ |     └── recovery.go                     # Интерсептор для ловли паники
│ │ ├── handler/                              # Слой хэндлеров (удовлетворяют grpc интерфейсу)
│ │ |     ├── grpc_analytics_integration.go   # Хэндлер для аналитики
│ │ |     ├── grpc_common.go                  # Хэндлер методы общего назначения
│ │ |     ├── grpc_organization.go            # Хэндлер для организации
│ │ |     ├── grpc_task_integration.go        # Хэндлер для задач
│ │ |     ├── grpc_telegram.go                # Хэндлер для телеграмма
│ │ |     ├── grpc_user_crud.go               # Хэндлер для пользователей
│ │ |     └── handler.go                      # Общее описание структуры хэндлера
│ │ ├── service/                              # Слой севиса (бизнесс-логика)
│ │ |     ├── analytics.go                    # Слой, отвечающий за работу с аналитикой (в рамках сервиса)
│ │ |     ├── organization.go                 # Слой, отвечающий за работу с организацией (в рамках сервиса)
│ │ |     ├── tasks.go                        # Слой, отвечающий за работу с пзадачами (в рамках сервиса)
│ │ |     ├── telegram.go                     # Слой, отвечающий за работу с телеграммом (в рамках сервиса)
│ │ |     ├── user.go                         # Слой, отвечающий за работу с пользователями (в рамках сервиса)
│ │ |     └── service.go                      # Общее описание структуры сервиса пользователей
│ │ ├── repository/                           # Слой репозитория (работа с БД)
│ │ |     ├── cache.go                        # Репозиторий кэша
│ │ |     ├── db_repo.go                      # Репозиторий БД (структура+конструктор)
│ │ |     ├── db_repo_org_methods.go          # Методы репозитория БД, относящиеся к организации
│ │ |     ├── db_repo_user_methods.go         # Методы репозитория БД, относящиеся к пользователю
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

```
| Поле                | Тип       | Описание                          |
| ------------------- | --------- | --------------------------------- |
| `id`                | UUID      | Первичный ключ                    |
| `name`              | string    | Название организации              |
| `owner_id`          | string    | ID владельца                      |
| `is_active`         | bool      | флаг активности                   |
| `created_at`        | timestamp | Дата создания                     |
```

### Пользователь (User)

```
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
```

### Роли и права

```
| Роль         | Права                                                                              |
| ------------ | ---------------------------------------------------------------------------------- |
| **owner**    | Всё: управление организацией, создание/удаление пользователей, просмотр всех задач |
| **manager**  | Создание задач, назначение исполнителей, просмотр задач организации                |
| **employee** | Только свои задачи, изменение статуса                                              |
```

## 🔐 Авторизация и аутентификация

### Архитектура авторизации

```
┌─────────────────┐
│   Telegram Bot  │ (внешний клиент)
└────────┬────────┘
         │
         │ 1. LinkTelegram (без токена)
         ▼
┌─────────────────────────────────────┐
│           User Service              │
│ ┌─────────────────────────────┐     │
│ │ gRPC Server                 │     │
│ │ ├── Public Methods (no auth)│     │
│ │ ├── JWT Interceptor         │     │
│ │ └── Service Interceptor     │     │
│ └─────────────────────────────┘     │
│                                     │
│ 🔥 ГЕНЕРАЦИЯ JWT (после привязки)   │
└────────┬───────────────┬────────────┘
         │               │
         ▼               ▼
    ┌─────────┐    ┌──────────┐
    │Postgres │    │   Redis  │
    └─────────┘    └──────────┘
        │               │
        │               │ JWT токен
        │               ▼
        │        ┌─────────────┐
        │        │Telegram Bot │
        │        │(сохраняет)  │
        │        └─────────────┘
        │
        │ API Key (для сервисов)
        ▼
┌─────────────────┐
│ Task Service    │ ──► вызывает user-service с API Key
└─────────────────┘
        │
        │ JWT (от Telegram бота)
        ▼
┌─────────────────┐
│ Task Service    │ ──► проверяет JWT локально (НЕ вызывает user-service)
└─────────────────┘
```

### Типы авторизации

```
| Тип запроса                             | Метод авторизации                | Где используется                                               |
| ------------------------------------    | -------------------------------- | -------------------------------------------------------------- |
| **Telegram → user-service** (привязка)  | Нет (публичный метод)            | `LinkTelegram`.                                                |
| **Telegram → user-service** (остальные) | JWT Bearer                       | `GetMyProfile`, `UpdateMyProfile`, `GetOrganization`,          |
|                                         |                                  | `CreateUser`, `GetUser`, `UpdateUser`, `DeleteUser`,           |
|                                         |                                  | `ListUsers`, `Logout`.                                         |
| **Telegram → task-service**             | JWT Bearer (проверяет локально)  | `CreateTask`, `UpdateStatus`, `GetMyTasks`.                    |
| **Сервис → user-service** (валидация)   | API Key (gRPC metadata)          | `ValidateToken`.                                               |
| **Сервис → user-service** (данные)      | API Key (gRPC metadata)          | `GetUserByID`, `GetUserByTelegram`, `BatchGetUsers`,           |
|                                         |                                  | `ValidateUser`, `CheckUserExists`, `GetUsersByRole`,           |
|                                         |                                  | `GetUserRole`, `GetAllUsers`.                                  |
| **Сервис → сервис**                     | API Key (gRPC metadata)          | task-service → user-service,                                   |
|                                         |                                  | analytics-service → user-service,                              |
|                                         |                                  | notification-service → user-service.                           |
| **Внутренние методы**                   | Нет (доверенные)                 | Миграции, воркеры, `SetupInitialOrganization`,                 |
|                                         |                                  | `HealthCheck`.                                                 |
```

### Поток генерации и использования JWT

```
┌─────────────────────────────────────────────────────────────────┐
│ ЭТАП 1: Привязка Telegram (без авторизации)                     │
├─────────────────────────────────────────────────────────────────┤
│        Telegram Bot ──(email, telegram_id)──► LinkTelegram      │
│                              │                                  │
│                              ▼                                  │
│                       User Service:                             │
│ 1. Находит пользователя                                         │
│ 2. Сохраняет telegram_id                                        │
│ 3. ГЕНЕРИРУЕТ JWT                                               │
│ 4. Сохраняет в Redis                                            │
│                              │                                  │
│                              ▼                                  │
│      Telegram Bot ◄────(JWT token)──── LinkTelegramRespon       │
│                              │                                  │
│                              ▼                                  │
│           Telegram Bot: Сохраняет JWT в свой Redis              │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ ЭТАП 2: Запросы с JWT (Telegram бот → User Service)             │
├─────────────────────────────────────────────────────────────────┤
│ Telegram Bot:                                                   │
│ 1. Сохраняет JWT в Redis (бот)                                  │
│ 2. Добавляет JWT в metadata: authorization: Bearer <jwt>        │
│ 3. Вызывает GetMyProfile / CreateUser / и т.д.                  │
│                              │                                  │
│                              ▼                                  │
│ User Service:                                                   │
│ 1. JWT Interceptor проверяет токен                              │
│ 2. Извлекает user_id, role, organization_id из JWT              │
│ 3. Проверяет права доступа                                      │
│ 4. Выполняет запрос                                             │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ ЭТАП 3: Запросы с API Key (Сервис → User Service)               │
├─────────────────────────────────────────────────────────────────┤
│ Task Service:                                                   │
│ 1. Добавляет API Key в metadata:                                │
│    x-service-name: task-service                                 │
│    x-service-key: tk_secret_123                                 │
│ 2. Вызывает GetUserByID / ValidateUser / и т.д.                 │
│                              │                                  │
│                              ▼                                  │
│ User Service:                                                   │
│ 1. Service Interceptor проверяет API Key                        │
│ 2. Выполняет запрос                                             │
└─────────────────────────────────────────────────────────────────┘
```

## gRPC API

### Сервис

```
protobuf
service UserService {
    // ==================== Публичные методы (без авторизации) ====================

    // SetupInitialOrganization - первичная настройка системы
    // Вызывается при первом запуске для создания организации и владельца
    rpc SetupInitialOrganization(SetupInitialOrganizationRequest) returns (SetupInitialOrganizationResponse);

    // LinkTelegram - привязка Telegram аккаунта
    // 🔥 ВОЗВРАЩАЕТ JWT токен для дальнейшей работы
    rpc LinkTelegram(LinkTelegramRequest) returns (LinkTelegramResponse);

    // HealthCheck - проверка здоровья сервиса
    rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);

    // ==================== Методы с JWT авторизацией ====================

    // GetMyProfile - получение своего профиля (JWT в metadata)
    rpc GetMyProfile(GetMyProfileRequest) returns (GetUserResponse);

    // UpdateMyProfile - обновление своего профиля (JWT в metadata)
    rpc UpdateMyProfile(UpdateMyProfileRequest) returns (GetUserResponse);

    // GetOrganization - получение информации об организации (JWT в metadata)
    rpc GetOrganization(GetOrganizationRequest) returns (GetOrganizationResponse);

    // Пользователи (CRUD) - требуют JWT + проверку прав
    rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);      // Только MANAGER/OWNER
    rpc GetUser(GetUserRequest) returns (GetUserResponse);               // OWNER/MANAGER видят всех, EMPLOYEE только себя
    rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);      // Разные права в зависимости от роли
    rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);      // Только OWNER
    rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);         // OWNER/MANAGER видят всех, EMPLOYEE только себя

    // Logout - выход из системы (отзыв JWT токена)
    rpc Logout(LogoutRequest) returns (LogoutResponse);

    // ==================== Методы для сервисов (API Key) ====================

    // ValidateToken - проверка JWT токена (для сервисов без JWT_SECRET)
    rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);

    // Интеграция с сервисом задач
    rpc ValidateUser(ValidateUserRequest) returns (ValidateUserResponse);           // Проверка пользователя для назначения
    rpc CheckUserExists(CheckUserExistsRequest) returns (CheckUserExistsResponse); // Быстрая проверка существования
    rpc GetUserByID(GetUserByIDRequest) returns (GetUserResponse);                 // Получение пользователя по ID
    rpc BatchGetUsers(BatchGetUsersRequest) returns (BatchGetUsersResponse);       // Массовое получение пользователей
    rpc GetUsersByRole(GetUsersByRoleRequest) returns (GetUsersByRoleResponse);    // Получение пользователей по роли

    // Интеграция с сервисом аналитики
    rpc GetAllUsers(GetAllUsersRequest) returns (GetAllUsersResponse);             // Все пользователи (для отчётов)
    rpc GetUserRole(GetUserRoleRequest) returns (GetUserRoleResponse);             // Быстрое получение роли

    // Интеграция с сервисом уведомлений
    rpc GetUserByTelegram(GetUserByTelegramRequest) returns (GetUserResponse);     // Поиск по Telegram ID
}
```

### Процесс регистрации и привязки Telegram

```
Этап 1: Создание организации и владельца (через SQL скрипт или API)

Этап 2: Привязка Telegram (через бота):
-------- 1. Пользователь пишет боту /start
-------- 2. Бот запрашивает email
-------- 3. Бот вызывает user-service.LinkTelegram(email, telegram_id)
-------- 4. User-service обновляет поле telegram_id
-------- 5. 🔥 User-service ГЕНЕРИРУЕТ JWT и возвращает боту
-------- 6. Бот сохраняет JWT в своём Redis для последующих запросов

Этап 3: Добавление сотрудников (через бота или API):
-------- 1. Руководитель через бота (/add_user) вводит email и имя
-------- 2. Бот вызывает user-service.CreateUser() (с JWT в metadata)
-------- 3. Создаётся пользователь с ролью employee, telegram_id = null
-------- 4. Сотрудник самостоятельно привязывает Telegram через /start
```

### Конфигурация

```
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
```

### Безопасность

Пароли — хранятся только в виде bcrypt-хеша

JWT — подписывается секретом, время жизни 24 часа

gRPC — в продакшене рекомендуется TLS

Rate limiting — через Redis

### Graceful Shutdown

Сервис корректно завершает работу:

1. Перестаёт принимать новые gRPC запросы
2. Завершает текущие запросы
3. Закрывает соединения с БД и Redis
4. Завершает процесс

### Health Check
