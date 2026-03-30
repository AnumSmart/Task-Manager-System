# Notification Service

Микросервис отправки уведомлений. Слушает события из Kafka и отправляет уведомления через различные каналы (Telegram, Email, Webhook).

## Технологии

- **Go 1.22**
- **Kafka** — приём событий
- **gRPC** — коммуникация с user-service (получение Telegram chat_id)
- **Telegram Bot API** — отправка сообщений

## Структура

notification-service/
├── cmd/
│ └── main.go # точка входа
├── internal/
│ ├── domain/ # бизнес-сущности
│ │ ├── notification.go # структура уведомления
│ │ ├── template.go # шаблоны сообщений
│ │ └── errors.go
│ ├── consumer/ # Kafka консьюмер
│ │ ├── kafka_consumer.go # подключение к Kafka
│ │ └── handler.go # обработка событий
│ ├── providers/ # провайдеры отправки
│ │ ├── provider.go # интерфейс NotificationProvider
│ │ ├── telegram.go # отправка в Telegram
│ │ ├── email.go # отправка email (опционально)
│ │ └── webhook.go # отправка webhook
│ ├── service/
│ │ ├── notification_service.go
│ │ └── template_service.go # форматирование сообщений
│ ├── client/
│ │ └── user_client.go # gRPC клиент user-service
│ └── config/
│ └── config.go
├── Dockerfile
├── go.mod
├── go.sum
└── .env

## Модель данных

### Уведомление (Notification)

```go
type Notification struct {
    ID          string
    UserID      string
    Type        string   // task_assigned, task_status_changed, task_overdue
    Title       string
    Message     string
    Channel     string   // telegram, email, webhook
    Status      string   // pending, sent, failed
    CreatedAt   time.Time
    SentAt      *time.Time
    Error       string
}
```

### Типы уведомлений

Тип ---------------------------------------- Событие ---------------------------------------- Пример
task_assigned -------------------------- Задача назначена ---------------- "📋 Вам назначена задача: Подготовить отчёт"
task_status_changed --------------------- Статус изменён --------------- "✅ Задача 'Отчёт' перешла в статус 'На проверке'"
task_completed ------------------------- Задача завершена ---------------------- "🎉 Задача 'Отчёт' выполнена!"
task_overdue -------------------------- Задача просрочена ---------------------- "⚠️ Задача 'Отчёт' просрочена!"
task_rejected -------------------------- Задача отклонена ----------------- "❌ Задача 'Отчёт' отклонена. Комментарий: ..."

### Kafka топики (consumer)

Топик ---------------------------------------- Событие ---------------------------------------- Действие
tasks.assigned -------------------------- TaskAssignedEvent ---------------------- Отправить уведомление исполнителю
tasks.status.changed ------------------ TaskStatusChangedEvent ------------ Отправить уведомление (кому — зависит от статуса)
tasks.completed ------------------------- TaskCompletedEvent -------------------- Уведомить руководителя и исполнителя
tasks.overdue ---------------------------- TaskOverdueEvent---------------------- Уведомить исполнителя и руководителя
tasks.rejected --------------------------- TaskRejectedEvent --------------------------- Уведомить исполнителя

### Конфигурация

Переменная ---------------------------------------- Описание ---------------------------------------- По умолчанию
KAFKA_BROKERS ---------------------------------- Kafka брокеры ------------------------------------- localhost:9092
KAFKA_GROUP_ID ------------------------------- Consumer group ID -------------------------------- notification-service
KAFKA_TOPICS -------------------------- Топики для подписки (через запятую) -------- tasks.assigned,tasks.status.changed,tasks.overdue
TELEGRAM_TOKEN ----------------------------- Токен Telegram бота ------------------------------------- обязательный
USER_SERVICE_GRPC --------------------------- Адрес user-service ----------------------------------- localhost:50052
LOG_LEVEL ---------------------------------- Уровень логирования ---------------------------------------- info

### Graceful Shutdown

Завершение работы:

1.  Останавливаем consumer (не принимаем новые сообщения)
2.  Дожидаемся обработки текущих сообщений
3.  Закрываем gRPC соединения
4.  Закрываем Kafka consumer
5.  Завершаем процесс

### Health Check
