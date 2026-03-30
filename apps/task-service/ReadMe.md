# Task Service

Микросервис управления задачами. Отвечает за создание, обновление, назначение и отслеживание статусов задач. Работает с PostgreSQL, Redis и Kafka.

## Технологии

- **Go 1.22**
- **PostgreSQL** — основное хранилище
- **Redis** — кэш, distributed locks
- **Kafka** — публикация событий
- **gRPC** — API для взаимодействия с другими сервисами

## Структура

task-service/
├── cmd/
│ └── main.go # точка входа
├── internal/
│ ├── domain/ # бизнес-сущности
│ │ ├── task.go # структура Task, методы
│ │ ├── status.go # статусы задачи
│ │ ├── priority.go # приоритеты
│ │ └── errors.go # domain errors
│ ├── repository/ # работа с БД
│ │ ├── task_repository.go # интерфейс
│ │ ├── assignee_repository.go
│ │ ├── history_repository.go
│ │ └── postgres/ # реализация на PostgreSQL
│ │ ├── db.go # подключение, транзакции
│ │ ├── task_repo_impl.go
│ │ ├── assignee_repo_impl.go
│ │ └── history_repo_impl.go
│ ├── service/ # бизнес-логика
│ │ ├── task_service.go # основная структура сервиса
│ │ ├── create_task.go # создание задачи (с сагой)
│ │ ├── update_task.go # обновление задачи
│ │ ├── reassign_task.go # переназначение
│ │ ├── complete_task.go # завершение задачи
│ │ └── query_task.go # получение задач
│ ├── server/
│ │ └── grpc_server.go # gRPC сервер
│ └── config/
│ └── config.go # конфигурация
├── api/
│ └── proto/
│ └── task.proto # gRPC схема
├── Dockerfile
├── go.mod
├── go.sum
└── .env

## Модель данных

### Основные сущности

| Сущность     | Описание                                                   |
| ------------ | ---------------------------------------------------------- |
| **Task**     | Задача: ID, название, описание, приоритет, дедлайн, статус |
| **Assignee** | Связь задачи с исполнителем (один ко многим)               |
| **History**  | Аудит изменений статусов                                   |

### Статусы задачи

| Статус        | Описание                       |
| ------------- | ------------------------------ |
| `new`         | Создана, ожидает назначения    |
| `assigned`    | Назначена исполнителю          |
| `in_progress` | В работе                       |
| `review`      | На проверке у руководителя     |
| `completed`   | Выполнена                      |
| `rejected`    | Отклонена, требуется доработка |
| `overdue`     | Просрочена                     |

### Приоритеты

- `low` — низкий
- `medium` — средний
- `high` — высокий

## gRPC API

### Сервис

```protobuf
service TaskService {
    // Создать задачу
    rpc CreateTask(CreateTaskRequest) returns (CreateTaskResponse);

    // Получить задачу по ID
    rpc GetTask(GetTaskRequest) returns (GetTaskResponse);

    // Обновить задачу
    rpc UpdateTask(UpdateTaskRequest) returns (UpdateTaskResponse);

    // Получить задачи пользователя (исполнителя)
    rpc GetUserTasks(GetUserTasksRequest) returns (GetUserTasksResponse);

    // Получить задачи организации (для руководителя)
    rpc GetOrganizationTasks(GetOrganizationTasksRequest) returns (GetOrganizationTasksResponse);

    // Изменить статус задачи
    rpc UpdateTaskStatus(UpdateTaskStatusRequest) returns (UpdateTaskStatusResponse);

    // Переназначить задачу
    rpc ReassignTask(ReassignTaskRequest) returns (ReassignTaskResponse);

    // Завершить задачу (принять работу)
    rpc CompleteTask(CompleteTaskRequest) returns (CompleteTaskResponse);
}
```

### События Kafka

Топик ---------- Событие ----------------------- Описание

tasks.created TaskCreatedEvent --------------- Задача создана
tasks.assigned TaskAssignedEvent ------------- Задача назначена
tasks.status.changed TaskStatusChangedEvent -- Статус изменён
tasks.completed TaskCompletedEvent ----------- Задача завершена
tasks.overdue TaskOverdueEvent --------------- Задача просрочена

### Saga создания задачи

При создании задачи выполняется распределённая транзакция:

1. Создать задачу в БД (status=new)
   ↓
2. Добавить связи с исполнителями
   ↓
3. Обновить статус → assigned
   ↓
4. Публикация события tasks.assigned (Kafka)

Если любой шаг (кроме 4) падает — транзакция откатывается. Шаг 4 не критичен, ошибки логируются.

### Конфигурация

Переменная ------------------------- Описание ------------------------- По умолчанию
DB_HOST --------------------------- PostgreSQL ----------------------- хост localhost
DB_PORT --------------------------- PostgreSQL -------------------------- порт 5432
DB_USER --------------------------- PostgreSQL ------------------- пользователь postgres
DB_PASSWORD ----------------------- PostgreSQL ----------------------- пароль postgres
DB_NAME --------------------------- PostgreSQL -------------------------- БД taskdb
REDIS_HOST -------------------------- Redis -------------------------- хост localhost
REDIS_PORT -------------------------- Redis ----------------------------- порт 6379
KAFKA_BROKERS ------------------- Kafka брокеры ---------------------- localhost:9092
GRPC_PORT ---------------------------- gRPC ---------------------------- порт 50051
USER_SERVICE_GRPC ------------- Адрес user-service ------------------- localhost:50052

### Graceful Shutdown

Сервис корректно завершает работу:

Перестаёт принимать новые gRPC запросы

Завершает текущие запросы

Закрывает соединения с БД, Redis, Kafka

Завершает процесс

### Health Check
