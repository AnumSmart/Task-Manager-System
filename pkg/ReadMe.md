# pkg - Shared Library for Task Management System

Этот пакет содержит переиспользуемые компоненты, общие для всех микросервисов проекта.

# структура пакета pkg

```
pkg/
├── auth/
│     ├── blacklist/
│     |    └── redis.go
|     ├── config/
│     |    ├── config.go
│     |    └── helper.go             # вспомогательные функции
|     ├── jwt/
│     |    ├── claims.go             # Только структуры claims
│     |    ├── errors.go             # JWT ошибки
│     |    └── manager.go            # JWT Manager (принимает конфиг)
|     ├── refresh/
|     |    └── storage.go            # Хранилище для refresh jwt ключей (сущность)
|     ├── interfaces.go
|     ├── auth.go
|     └── provider.go                # DI провайдер
|
├── config/
│     ├── config_loader_yml.go       # универсальная функция для загрузки конфига из .yml файла
│     ├── grpc_server_config.go      # универсальный конфиг и конструктор для grpc сервера
│     ├── event_consumer_config.go   # универсальный конфиг и конструктор event consumer
│     ├── event_publisher_config.go  # универсальный конфиг и конструктор event publisher
│     ├── jwt_config.go              # универсальный конфиг и конструктор для логики авторизации
│     ├── outbox_relay_config.go     # универсальный конфиг для реле outbox
│     ├── postgres_config.go         # универсальный конфиг для экземпляра БД POSTGRES
│     ├── rabbit_mq_config.go        # универсальный конфиг для экземпляра rabbitMQ
│     ├── redis_config.go            # универсальный конфиг для экземпляра БД REDIS
|     └── tools.go                   # набор универсальных функций
|
├── db/
│     ├── db.go                      # конструктор, который создаёт пул соединений, который соотетствует глобальному интерфейсу
│     └── pool_adapter.go            # адаптер, чтобы новый пул соответствовал глобальному интерфейсу
├── eventbus/                        # описание логики работы с событиями (для переиспользования)
│     ├── consumer.go                # логика работы event cosumer
│     ├── event_registry.go          # логика реестра событий (чтобы различать какое событие пришло)
│     ├── handler_registry.go        # логика работы реестра хэндлеров (для потребления событий)
│     ├── interface.go               # интерфейсы для eventbus
│     ├── middleware.go              # логирование и ловля паник
│     └── publisher.go               # логика eventPublisher
├── events/                          # описание события (для переиспользования)
│     └── base_event.go              # базовое описание структруы события и интерфейс
├── rabbitmq/                        # описание работы с rabbitmq
│     ├── broker_states.go           # определение состояний брокера (connected/disconnected/closing/connecting)
│     ├── broker.go                  # основная структура Broker с полями (conn, channel, mutex'ы)
│     ├── connecion.go               # управление жизненным циклом соединения с RabbitMQ
│     ├── consume.go                 # реализация получения сообщений из очереди
│     ├── errors.go                  # общие ошибки пакета rabbitmq
│     ├── health.go                  # методы для мониторинга состояния брокера
│     ├── interface.go               # публичный интерфейс брокера (BrokerInterface)
│     ├── publish.go                 # реализация отправки сообщений в RabbitMQ
│     └── retry.go                   # умная обработка ошибок с повторными попытками
├── logger/                          # описание работы с логгером (пока в разоаботке)
├── outbox/                          # логика работы outbox
│     ├── interface.go               # интерфейс работы с outbox таблицей
│     ├── models.go                  # описание модели outboxMessage
│     ├── postgres_outbox_repo.go    # репозиторий для работы с outbox (на базе Postgres)
│     └── relay.go                   # логика ralay outbox
├── redis/
│     ├── redis.go                   # конструктор, который создаёт экземпляр redis, который соотетствует глобальному интерфейсу
│     └── pool_adapter.go            # адаптер, чтобы экземпляр redis соответствовал глобальному интерфейсу
├── go.mod
├── go.sum
└── ReadMe.md
```
