# структура auth

```
pkg/auth/
├── auth.go                    # Основная логика
├── interfaces.go              # Интерфейсы
├── provider.go                # DI провайдер
├── jwt/
│   ├── claims.go              # Структуры claims
│   ├── errors.go              # JWT ошибки
│   └── manager.go             # JWT Manager (только JWT логика)
├── blacklist/
│   └── blacklist.go           # Интерфейс и логика blacklist (без создания клиента)
└── refresh/
    └── storage.go             # Хранилище для refresh jwt ключей (сущность)

```
