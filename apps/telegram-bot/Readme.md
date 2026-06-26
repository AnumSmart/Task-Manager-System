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
│   └── bot/
│       └── main.go                          # Точка входа
│
├── internal/
│   ├── app/                                 # Сборка приложения
│   │   ├── container.go                     # DI контейнер
│   │   ├── bot.go                          # Запуск/остановка бота
│   │   └── shutdown.go                     # Graceful shutdown
│   │
│   ├── config/                              # Конфигурация
│   │   └── config.go
│   │
│   ├── handlers/                            # Обработчики команд
│   │   ├── auth/
│   │   │   ├── start.go                    # /start - привязка Telegram
│   │   │   └── middleware.go               # Проверка JWT сессии
│   │   ├── tasks/
│   │   │   ├── list.go                     # /tasks - мои задачи
│   │   │   ├── list_all.go                 # /tasks_all - все задачи (только менеджер/owner)
│   │   │   ├── create.go                   # /create - создание задачи (диалог)
│   │   │   ├── status.go                   # /status - изменение статуса
│   │   │   └── assign.go                   # /assign - назначить исполнителя
│   │   ├── callback.go                     # Обработка inline-кнопок
│   │   └── router.go                       # Маршрутизация команд
│   │
│   ├── service/                             # Бизнес-логика
│   │   ├── auth_service.go                 # Логика привязки/аутентификации
│   │   ├── task_service.go                 # Логика работы с задачами
│   │   ├── user_service.go                 # Логика работы с пользователями
│   │   └── service.go                      # Общий интерфейс
│   │
│   ├── client/                              # gRPC клиенты
│   │   ├── task_client.go                  # Клиент для task-service
│   │   ├── user_client.go                  # Клиент для user-service
│   │   ├── auth_interceptor.go             # JWT interceptor для gRPC
│   │   └── client.go                       # Общий клиент
│   │
│   ├── session/                             # Управление сессиями (Redis)
│   │   ├── manager.go                      # Интерфейс менеджера
│   │   ├── redis.go                        # Redis реализация
│   │   ├── models.go                       # Модель сессии (с JWT)
│   │   └── keys.go                         # Ключи для Redis
│   │
│   ├── keyboard/                            # Клавиатуры
│   │   ├── main_menu.go                    # Главное меню
│   │   ├── task_menu.go                    # Меню задач
│   │   ├── status_menu.go                  # Меню статусов
│   │   └── inline.go                       # Inline-кнопки
│   │
│   ├── formatter/                           # Форматирование сообщений
│   │   ├── task.go                         # Форматирование задач
│   │   ├── user.go                         # Форматирование пользователей
│   │   └── error.go                        # Форматирование ошибок
│   │
│   └── dto/                                 # Data Transfer Objects
│       ├── request.go                      # Запросы к сервисам
│       └── response.go                     # Ответы от сервисов
│
├── pkg/                                     # Переиспользуемые пакеты
│   ├── logger/
│   ├── errors/
│   └── jwt/                                 # JWT утилиты (если нужны)
│
├── configs/                                 # Конфиги
│   ├── config.yml
│   └── config_dev.yml
│
├── migrations/                              # Если бот использует свою БД
├── Dockerfile
├── go.mod
├── go.sum
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
