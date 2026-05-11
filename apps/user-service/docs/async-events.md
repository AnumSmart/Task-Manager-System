# Асинхронные события user-service

## Обзор

User-service публикует события о изменениях в доменной модели (пользователи, организации,
роли) в RabbitMQ. Другие сервисы (notification, task, analytics) потребляют эти события
для асинхронной реакции.

---

## Топология RabbitMQ

| Параметр          | Значение                                      |
| ----------------- | --------------------------------------------- |
| Exchange name     | `taskman.events`                              |
| Exchange type     | `topic`                                       |
| Routing key       | `user.{event_type}` (например `user.created`) |
| Владелец очередей | Сервисы-потребители (не user-service)         |

> **Примечание:** user-service НЕ создаёт очереди. Очереди создают сервисы-потребители
> и привязывают их к exchange с нужными routing_key. User-service только публикует.

---

## Формат сообщения (общий)

```json
{
	"event_id": "550e8400-e29b-41d4-a716-446655440000",
	"event_type": "user.{event_type}",
	"service": "user-service",
	"version": 1,
	"timestamp": "2025-01-15T10:30:00Z",
	"data": {}
}
```

## Список событий

### 1. `user.created`

**Триггер:** `CreateUser` Доменный метод

**Routing key:** `user.created`

**Схема:**

```json
{
	"event_id": "550e8400-e29b-41d4-a716-446655440000",
	"event_type": "user.created",
	"service": "user-service",
	"version": 1,
	"timestamp": "2025-01-15T10:30:00Z",
	"data": {
		"user_id": "550e8400-e29b-41d4-a716-446655440000",
		"organization_id": "660e8400-e29b-41d4-a716-446655440001",
		"email": "user@example.com",
		"full_name": "Иван Иванов",
		"role": "EMPLOYEE",
		"status": "ACTIVE"
	}
}
```

### 2. `user.telegram_linked`

**Триггер:** `LinkTelegram` (доменный метод)

**Routing key:** `user.telegram_linked`

**Схема:**

```json
{
	"event_id": "550e8400-e29b-41d4-a716-446655440000",
	"event_type": "user.created",
	"service": "user-service",
	"version": 1,
	"timestamp": "2025-01-15T10:30:00Z",
	"data": {
		"user_id": "550e8400-e29b-41d4-a716-446655440000",
		"organization_id": "660e8400-e29b-41d4-a716-446655440001",
		"telegram_id": 123456789,
		"email": "user@example.com"
	}
}
```
