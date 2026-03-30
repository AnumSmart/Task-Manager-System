# Apps

Микросервисы системы управления поручениями.

## Структура

apps/
├── telegram-bot/ # Telegram бот — основной интерфейс
├── task-service/ # Управление задачами и статусами
├── user-service/ # Организации, пользователи, роли
├── notification-service/ # Отправка уведомлений (Telegram)
└── analytics-service/ # Сбор метрик и формирование отчётов

## Сервисы

| Сервис                   | Описание                                                      | Порт (gRPC) | Основные технологии      |
| ------------------------ | ------------------------------------------------------------- | ----------- | ------------------------ |
| **telegram-bot**         | Интерфейс пользователя, команды `/start`, `/tasks`, `/create` | —           | Gin, long polling        |
| **task-service**         | CRUD задач, статусная модель, Saga                            | 50051       | PostgreSQL, Redis, Kafka |
| **user-service**         | Организации, пользователи, роли, привязка Telegram            | 50052       | PostgreSQL, bcrypt, JWT  |
| **notification-service** | Получение событий из Kafka, отправка через Telegram           | —           | Kafka, Telegram API      |
| **analytics-service**    | Агрегация метрик, отчёты                                      | 50053       | Kafka, PostgreSQL        |

## Взаимодействие

telegram-bot ──gRPC──► task-service ──Kafka──► notification-service
│ │ │
│ │ │
└──gRPC──► user-service◄─────────────────────────┘
│
▼
analytics-service ◄──Kafka──┘

### Общие зависимости

Все сервисы используют:

pkg/ — общие библиотеки (модели, Kafka, gRPC, логгер)

api/proto/ — protobuf схемы

migrations/ — миграции БД (goose)
