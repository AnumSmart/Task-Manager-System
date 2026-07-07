# Telegram Bot

Telegram-бот для системы управления поручениями. Работает через long polling (getUpdates).

## Технологии

- **Go 1.22**
- **go-telegram-bot-api** — работа с Telegram API
- **gRPC** — коммуникация с task-service и user-service
- **Long polling** — приём обновлений (без webhook)

## Структура

```
telegram-bot/
├── cmd/
│   ├── di_container.go                        # Создание DI контейнера
│   ├── grace_shut_down.go                     # Описание функций gracefull shutdown
│   ├── health_check.go                        # Описание функций health check
│   ├── start_server.go                        # Описание функций запуска сервера
│   └── main.go                                # точка входа
│
├── internal/
│   ├── config/                             # Конфигурация
│   |   ├── app_config.go                      # Общий конфиг для всего приложения
│   |   ├── botConfig.go                       # Конфиг для бота
│   |   ├── botServerConfig.go                 # Конфиг для сервера
│   │   └── grpcClientConfig.go                # Конфиг для grpc клиента
│   ├── deps/                               # работа с зависимостями
│   │   ├── di.go                              # описание DI контейнера
│   │   └── di_methods.go                      # методы DI контейнера
│   ├── domain/                             # бизнес-сущности
│   │   └── user.go                            # модели для работы с сервисом user-service
│   ├── grpc_clients/                       # логика клиентов для работы со сторонними сервисами по grpc
│   │   └── grpc_user_client/                # логика grpc клиента для работы с user-service
│   │        |── client.go                     # логика grpc клиента
│   │        |── interfaces.go                 # интерфейсы для тестирования
│   │        └── mapper.go                     # маппер grpc моделей в доменнык
│   ├── server/                             # логика http сервера
│   │   ├── handlers/                        # логика хэндлеров
│   │   |    └── bot_handlers.go               # хэндлеры для работы с телеграмм ботом
│   │   ├── reg_bot_handlers.go                # логика регистрации хэндлеров
│   │   ├── bot_polling.go                     # логика управления long polling ботом
│   │   └── server.go                          # http server
├── yml-configs/
│   ├── botConfig.yml                          # Конфигурация бота
│   ├── botServerConfig.yml                    # Конфигурация сервера
│   ├── grpcUserClientConfig.yml               # Конфигурация grpc клиента
│   └── logger.yml                             # Конфигурация логгера
├── go.mod
├── go.sum
├── Readme.md
└── .env                                     # Секреты (токены, пароли)
```

## 🔐 Управление JWT сессиями

### Структура сессии в Redis

```
json
{
  "chat_id": 123456789,
  "jwt_token": "eyJhbGciOiJIUzI1NiIs...",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "admin@romashka.ru",
  "role": "owner",
  "org_id": "660e8400-e29b-41d4-a716-446655440000",
  "full_name": "Иван Иванов",
  "expires_at": "2024-01-15T12:00:00Z",
  "created_at": "2024-01-14T12:00:00Z",
  "last_active": "2024-01-14T15:30:00Z"
}
```

## Команды бота

```
| Команда      | Описание                                  | Доступ       |
| ------------ | ----------------------------------------- | ------------ |
| `/start`     | Регистрация, привязка Telegram к аккаунту | Все          |
| `/tasks`     | Список моих задач                         | Исполнитель  |
| `/tasks_all` | Список всех задач (с фильтрами)           | Руководитель |
| `/create`    | Создать новую задачу                      | Руководитель |
| `/status`    | Изменить статус задачи                    | Исполнитель  |
| `/help`      | Помощь                                    | Все          |
```
