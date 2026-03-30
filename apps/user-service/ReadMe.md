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

user-service/
├── cmd/
│ └── main.go # точка входа
├── internal/
│ ├── domain/ # бизнес-сущности
│ │ ├── organization.go # структура Organization
│ │ ├── user.go # структура User
│ │ ├── role.go # роли: owner, manager, employee
│ │ └── errors.go # domain errors
│ ├── repository/ # работа с БД
│ │ ├── organization_repository.go
│ │ ├── user_repository.go
│ │ └── postgres/
│ │ ├── db.go # подключение, транзакции
│ │ ├── organization_repo_impl.go
│ │ └── user_repo_impl.go
│ ├── service/ # бизнес-логика
│ │ ├── user_service.go # CRUD пользователей
│ │ ├── auth_service.go # регистрация, логин, JWT
│ │ ├── organization_service.go
│ │ └── telegram_service.go # привязка Telegram
│ ├── server/
│ │ └── grpc_server.go # gRPC сервер
│ └── config/
│ └── config.go
├── api/
│ └── proto/
│ └── user.proto # gRPC схема
├── Dockerfile
├── go.mod
├── go.sum
└── .env

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
    // Аутентификация
    rpc Register(RegisterRequest) returns (RegisterResponse);
    rpc Login(LoginRequest) returns (LoginResponse);
    rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);

    // Пользователи
    rpc GetUser(GetUserRequest) returns (User);
    rpc GetUserByEmail(GetUserByEmailRequest) returns (User);
    rpc GetUserByTelegramChatID(GetUserByTelegramChatIDRequest) returns (User);
    rpc ListOrganizationUsers(ListOrganizationUsersRequest) returns (ListOrganizationUsersResponse);
    rpc CreateUser(CreateUserRequest) returns (User);
    rpc UpdateUser(UpdateUserRequest) returns (User);
    rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);

    // Telegram привязка
    rpc LinkTelegram(LinkTelegramRequest) returns (LinkTelegramResponse);
    rpc GetTelegramChatID(GetTelegramChatIDRequest) returns (GetTelegramChatIDResponse);

    // Организации
    rpc GetOrganization(GetOrganizationRequest) returns (Organization);
    rpc UpdateOrganization(UpdateOrganizationRequest) returns (Organization);
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
