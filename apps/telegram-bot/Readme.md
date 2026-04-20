# Telegram Bot

Telegram-бот для системы управления поручениями. Работает через long polling (getUpdates).

## Технологии

- **Go 1.22**
- **go-telegram-bot-api** — работа с Telegram API
- **gRPC** — коммуникация с task-service и user-service
- **Long polling** — приём обновлений (без webhook)

## Структура

telegram-bot/
├── cmd/
│ └── main.go
├── internal/
│ ├── handlers/
│ │ ├── start.go # /start — регистрация
│ │ ├── tasks.go # /tasks — список задач
│ │ ├── create.go # создание задачи (пошаговый диалог)
│ │ ├── status.go # изменение статуса задачи
│ │ └── callback.go # обработка inline-кнопок
│ ├── keyboard/
│ │ └── menus.go # клавиатуры
│ ├── client/
│ │ ├── task_client.go # gRPC клиент для task-service
│ │ └── user_client.go # gRPC клиент для user-service
│ └── session/
│ └── manager.go # управление состояниями диалогов
├── go.mod
├── Dockerfile
└── .env

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

| Команда      | Описание                                  | Доступ       |
| ------------ | ----------------------------------------- | ------------ |
| `/start`     | Регистрация, привязка Telegram к аккаунту | Все          |
| `/tasks`     | Список моих задач                         | Исполнитель  |
| `/tasks_all` | Список всех задач (с фильтрами)           | Руководитель |
| `/create`    | Создать новую задачу                      | Руководитель |
| `/status`    | Изменить статус задачи                    | Исполнитель  |
| `/help`      | Помощь                                    | Все          |
