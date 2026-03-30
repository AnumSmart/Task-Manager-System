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

## Команды бота

| Команда      | Описание                                  | Доступ       |
| ------------ | ----------------------------------------- | ------------ |
| `/start`     | Регистрация, привязка Telegram к аккаунту | Все          |
| `/tasks`     | Список моих задач                         | Исполнитель  |
| `/tasks_all` | Список всех задач (с фильтрами)           | Руководитель |
| `/create`    | Создать новую задачу                      | Руководитель |
| `/status`    | Изменить статус задачи                    | Исполнитель  |
| `/help`      | Помощь                                    | Все          |
