# User Service

Микросервис управления пользователями, организациями и ролями.
Отвечает за аутентификацию, авторизацию, привязку Telegram и управление сотрудниками.
Генерирует JWT токены, но НЕ хранит их (кроме refresh токенов в Redis).

## Технологии

- **Go 1.22**
- **PostgreSQL** — основное хранилище
- **Redis** — хранение refresh токенов, blacklist
- **gRPC** — API для взаимодействия с другими сервисами
- **JWT** — аутентификация (stateless)
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
│ │ |     ├── api_key_auth.go                 # Интерсептор для внутренней межсервисной авторизации (api key)
│ │ |     ├── jwt_auth.go                     # Интерсептор для авторизации (jwt)
│ │ |     ├── method_classifier.go            # Классификатор методов для использования интерсептором
│ │ |     ├── public.go                       # Интерсептор для логирования вызова public методов
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
┌─────────────────────────────────────────────────────────────────┐
│                         Клиентский слой                         │
├─────────────────────────────────────────────────────────────────┤
│                      Telegram Bot                               │
│              ┌────────────────────────────┐                     │
│              │  Хранит токены в ПАМЯТИ:   │                     │
│              │  • Access token (JWT)      │                     │
│              │  • Refresh token           │                     │
│              └────────────────────────────┘                     │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ 1. LinkTelegram (без токена)
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                        User Service                             │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ 🔥 ГЕНЕРАЦИЯ JWT (после привязки Telegram)                │  │
│  │ • Access token (15 min) — stateless                       │  │
│  │ • Refresh token (7 days) — хранится в Redis               │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  Хранилище в Redis:                                             │
│  • refresh:{session_id} → {token_hash} (7 days TTL)        │
│  • blacklist:{token_hash} → expiry (до истечения access token)  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ Возвращает access + refresh
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Telegram Bot                             │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ ✅ Сохраняет оба токена в памяти                          │  │
│  │ ❌ НЕ хранит в БД (при рестарте сессия теряется)          │  │
│  └───────────────────────────────────────────────────────────┘  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ 2. Запрос с JWT (Bearer token)
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Task Service                             │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ ✅ Проверяет JWT ЛОКАЛЬНО (через общий JWT_SECRET)        │  │
│  │ ❌ НЕ вызывает user-service для валидации                 │  │
│  │ ✅ Проверяет blacklist (опционально, через Redis)         │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
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
│ Telegram Bot ──(email, telegram_id)──► LinkTelegram             │
│                              │                                  │
│                              ▼                                  │
│ User Service:                                                   │
│ 1. Находит/создаёт пользователя                                 │
│ 2. Сохраняет telegram_id                                        │
│ 3. 🔥 ГЕНЕРИРУЕТ access token (JWT, 15 min)                     │
│ 4. 🔥 ГЕНЕРИРУЕТ refresh token (7 days)                         │
│ 5. Сохраняет refresh token в Redis (hash)                       │
│                              │                                  │
│                              ▼                                  │
│ Telegram Bot ◄────(access + refresh)──── LinkTelegramResponse   │
│                              │                                  │
│                              ▼                                  │
│ Telegram Bot:                                                   │
│ ✅ Сохраняет access token в памяти                              │
│ ✅ Сохраняет refresh token в памяти                             │
│ ❌ НЕ сохраняет в БД                                            │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ ЭТАП 2: Истечение access token (через 15 минут)                 │
├─────────────────────────────────────────────────────────────────┤
│ Telegram Bot:                                                   │
│ 1. Обнаруживает, что access token истёк                         │
│ 2. Берёт refresh token из памяти                                │
│ 3. Вызывает user-service.RefreshToken(refresh_token)            │
│                              │                                  │
│                              ▼                                  │
│ User Service:                                                   │
│ 1. Проверяет refresh token в Redis                              │
│ 2. Генерирует НОВЫЙ access token                                │
│ 3. Отправляет новый access token                                │
│                              │                                  │
│                              ▼                                  │
│ Telegram Bot:                                                   │
│ ✅ Обновляет access token в памяти                              │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ ЭТАП 3: Запрос с JWT (Telegram → Task Service)                  │
├─────────────────────────────────────────────────────────────────┤
│ Telegram Bot:                                                   │
│ 1. Берёт access token из памяти                                 │
│ 2. Добавляет в metadata: authorization: Bearer <jwt>            │
│ 3. Вызывает task-service.CreateTask()                           │
│                              │                                  │
│                              ▼                                  │
│ Task Service:                                                   │
│ 1. ✅ Проверяет JWT ЛОКАЛЬНО (через общий JWT_SECRET)           │
│ 2. ❌ НЕ вызывает user-service для валидации                    │
│ 3. Извлекает user_id из claims                                  │
│ 4. Выполняет запрос                                             │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ ЭТАП 4: Выход из системы (Logout)                               │
├─────────────────────────────────────────────────────────────────┤
│ Telegram Bot:                                                   │
│ 1. Вызывает user-service.Logout(access_token)                   │
│                              │                                  │
│                              ▼                                  │
│ User Service:                                                   │
│ 1. Удаляет refresh token из Redis                               │
│                              │                                  │
│                              ▼                                  │
│ Telegram Bot:                                                   │
│ ✅ Удаляет оба токена из памяти                                 │
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
    // 🔥 ВОЗВРАЩАЕТ JWT токены для дальнейшей работы
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
Этап 1: Создание организации и владельца
─────────
Через миграцию или SetupInitialOrganization

Этап 2: Привязка Telegram (через бота)
─────────
1. Пользователь пишет боту /start
2. Бот запрашивает email
3. Бот вызывает user-service.LinkTelegram(email, telegram_id)
4. User-service:
   - Обновляет поле telegram_id
   - 🔥 ГЕНЕРИРУЕТ access_token (15 min)
   - 🔥 ГЕНЕРИРУЕТ refresh_token (7 days)
   - Сохраняет refresh_token в Redis (хеш, 7 дней)
5. User-service возвращает оба токена
6. Бот ✅ СОХРАНЯЕТ токены в ПАМЯТИ (мапа в RAM)
7. Бот отвечает пользователю "✅ Аккаунт привязан"

Этап 3: Использование API
─────────
1. Пользователь пишет /my_tasks
2. Бот берёт access_token из памяти
3. Бот вызывает task-service с JWT в metadata
4. Task-service проверяет JWT локально (через общий секрет)
5. Task-service возвращает задачи
6. Бот показывает их пользователю

Этап 4: Обновление истёкшего access_token
─────────
1. При истечении access_token (15 min)
2. Бот обнаруживает ошибку 401 Unauthorized
3. Бот берёт refresh_token из памяти
4. Бот вызывает user-service.RefreshToken(refresh_token)
5. User-service проверяет refresh_token в Redis
6. User-service генерирует НОВЫЙ access_token
7. Бот обновляет access_token в памяти

Этап 5: Выход из системы
─────────
1. Пользователь пишет /logout
2. Бот вызывает user-service.Logout(access_token, refresh_token)
3. User-service:
   - Добавляет access_token в blacklist (Redis, TTL до expiry)
   - Удаляет refresh_token из Redis
4. Бот удаляет оба токена из памяти
```

### Что хранится в Redis в User Service

```
--- Refresh tokens (хеш, для безопасности)
refresh:{user_id}:{token_hash} -> user_id  (7 days TTL)

--- Blacklist (отозванные access токены)
blacklist:{token_hash} -> expiry_timestamp  (TTL = осталось жить токену)
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
