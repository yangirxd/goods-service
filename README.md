# Goods Service

Сервис для управления товарами с поддержкой приоритезации, кэширования в Redis и логированием в ClickHouse.

## Быстрый старт

1. Клонируйте репозиторий
2. Запустите все сервисы через Docker Compose:
```bash
docker-compose up -d
```

Сервис будет доступен по адресу: http://localhost:8080

## Структура проекта

```
├── cmd/
│   └── main.go          # Точка входа в приложение
├── internal/
│   ├── cache/           # Работа с Redis
│   ├── clickhouse/      # Работа с ClickHouse
│   ├── db/             # Работа с PostgreSQL
│   ├── handler/        # HTTP обработчики
│   ├── models/         # Модели данных
│   ├── queue/          # Работа с NATS
│   └── repository/     # Репозиторий для работы с БД
├── migrations/
│   ├── clickhouse/     # Миграции ClickHouse
│   └── postgres/       # Миграции PostgreSQL
├── docker-compose.yml
└── Dockerfile
```

## API Endpoints

### Получение списка товаров
```http
GET /goods/list?limit=10&offset=0

Response:
{
    "meta": {
        "total": 100,    // общее количество записей
        "removed": 5,    // количество удалённых записей
        "limit": 10,     // текущий лимит
        "offset": 0      // текущее смещение
    },
    "goods": [...]
}
```

### Создание товара
```http
POST /goods/create
Content-Type: application/json

{
    "project_id": 1,
    "name": "Товар",
    "description": "Описание"
}
```

### Получение товара
```http
GET /goods/get/:id
```

### Обновление товара
```http
PATCH /goods/update/:id
Content-Type: application/json

{
    "name": "Новое название",
    "description": "Новое описание"
}
```

### Изменение приоритета
```http
PATCH /goods/reprioritize?id=123&projectId=456
Content-Type: application/json

{
    "newPriority": 5
}

Response:
{
    "priorities": [
        {"id": 123, "priority": 5},
        {"id": 124, "priority": 6},
        {"id": 125, "priority": 7}
    ]
}
```

### Удаление товара
```http
DELETE /goods/delete/:id
```

## Тестирование API

1. Создайте несколько товаров:
```bash
# Создаем первый товар
curl -X POST http://localhost:8080/goods/create \
    -H "Content-Type: application/json" \
    -d '{"project_id": 1, "name": "Товар 1", "description": "Описание 1"}'

# Создаем второй товар
curl -X POST http://localhost:8080/goods/create \
    -H "Content-Type: application/json" \
    -d '{"project_id": 1, "name": "Товар 2", "description": "Описание 2"}'

# Создаем третий товар
curl -X POST http://localhost:8080/goods/create \
    -H "Content-Type: application/json" \
    -d '{"project_id": 1, "name": "Товар 3", "description": "Описание 3"}'
```

2. Проверьте список товаров:
```bash
curl http://localhost:8080/goods/list
```

3. Измените приоритет товара:
```bash
curl -X PATCH "http://localhost:8080/goods/reprioritize?id=2&projectId=1" \
    -H "Content-Type: application/json" \
    -d '{"newPriority": 5}'
```

4. Проверьте изменение приоритетов:
```bash
curl http://localhost:8080/goods
```

## Мониторинг

- Redis доступен на порту 6379
- PostgreSQL доступен на порту 5432
- ClickHouse доступен на порту 9000
- NATS доступен на порту 4222

Логи операций сохраняются в ClickHouse и доступны через запрос:
```sql
SELECT * FROM logs.goods_log ORDER BY timestamp DESC;
```

## Остановка сервиса

Для остановки всех сервисов выполните:
```bash
docker-compose down
```

Для остановки с удалением всех данных:
```bash
docker-compose down -v
```
