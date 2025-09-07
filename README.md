# task_queue

Этот проект представляет собой небольшой сервис для приёма и фоновой обработки задач с возможностью повторов при ошибках.

## Структура проекта

```text
|-- cmd                                 # entrypoint
|   `-- main.go
|-- go.mod                              # модуль
|-- internal
|   |-- adapter
|   |   |-- rest
|   |   |   |-- server.go               # методы Run и Stop для сервера
|   |   |   `-- task_controller.go      # ручки
|   |   `-- workerpool
|   |       `-- workerpool.go           # worker pool и методы для работы с ним + retry/backoff механизм
|   |-- model
|   |   |-- create_task_request.go      # DTO для создания задачи
|   |   `-- task.go                     # модель задачи
|   |-- repository
|   |   `-- inmemory
|   |       `-- task_repository.go      # in-memory репозиторий для хранения задач (CRUD)
|   `-- usecase
|       `-- task_service.go             # сервисный слой + имитация работы таски
`-- pkg
    `-- apperrors
        `-- apperrors.go                # обертки над ошибками
```

## Возможности

- Создание задачи.
- Передача задачи в буферезированную внутреннюю очередь.
- Ассинхронная обработка задач.

## Особенности

- Чистая архитектура.
- REST-ful API без использования сторонних фреймворков и роутеров.
- Пайплайн работы: `POST /enqueue -> Save(service -> repository) & PushToQueue(worker_pool) -> worker(worker_pool) -> HandleTask(service) if success -> { status=done } else { for max_retries && status!=done { backoff + jitter -> PushToQueue(worker_pool) } }`.
- Таски хранятся в мапе, защищенной от параллельного доступа к данным RWMutex.
- Конфигурационные переменные инициализируются из переменных окружения. В случае если таковы не заданы, принимают дефолтные значения.
- usecase-слой и кэш покрыты тестами. ////
- DTO структура для того чтобы не принять лишних полей из запроса на создание. Лишние могут появится, так как в модель задачи были добавлены поля Attempts (для подсчета предпринятых попыток) и Status (для отслеживания состояния заказа).
- В работе worker pool реализован механизм retry/backoff (отдельная retry-queue с воркером) c экспоненциальным увеличением времени в зависимости от номера попытки и jitter, принимающий значения от 0 до половины от минимальной задержки (100ms).
- graceful shutdown: работает по принципу прослушивания сигналов SIGTERM и SIGINT; после отработки запускается flow отмены контекста и закрытия каналов; для того, чтобы задачи могли завершиться корректно используется WaitGroup в каждом из воркеров (и в retry воркере тоже).
- Были реализованы дополнительно ручки `GET /tasks` и `GET /task?id=<task_id>` для дебага.
- Ручка `GET /healthz` служит датчиком жизни сервера.

## Запуск и тестирование

1. Конфигурация переменных окружения

```shell
export QUEUE_SIZE=64 # default=64
export WORKERS=4     # default=4
```

2. Тестирование (unit, integration)

```shell
go test ./... -v
```

3. Запуск программы

```shell
go run cmd/main.go
```

4. Тестирование (postman/curl)

### `POST /enqueue`

Добавить новую задачу в очередь.

*request*

```json
{
  "id": "task-123",
  "payload": "some data",
  "max_retries": 3
}
```

*response*

`201 Created` — задача успешно принята:

```json
{
  "id": "task-123",
  "payload": "some data",
  "max_retries": 3,
  "status": "queued",
  "attempts": 0
}
```

`400 Bad Request` — некорректный JSON или данные:

```json
{
  "error": "invalid JSON"
}
```

`409 Conflict` — задача с таким ID уже существует:

```json
{
  "error": "task already exists"
}
```

---

### `GET /healthz`

Проверка состояния сервиса.

*request*

```text
empty
```

*response*

`200 OK` — сервис работает:

```json
{
  "status": "ok"
}
```

---

### `GET /task?id=<task_id>`

Получить информацию о конкретной задаче по её ID.

*request*

```text
id=task-123
```

*response*

`200 OK` — задача найдена:

```json
{
  "id": "task-123",
  "payload": "some data",
  "max_retries": 3,
  "status": "running",
  "attempts": 1
}
```

`400 Bad Request` — отсутствует параметр `id`:

```json
{
  "error": "missing id parameter"
}
```

`404 Not Found` — задача не найдена:

```json
{
  "error": "task not found"
}
```

---

### `GET /tasks`

Получить список всех задач.

*request*

```text
empty
```

*response*

`200 OK` — массив задач:

```json
[
  {
    "id": "task-123",
    "payload": "some data",
    "max_retries": 3,
    "status": "done",
    "attempts": 1
  },
  {
    "id": "task-124",
    "payload": "other data",
    "max_retries": 2,
    "status": "failed",
    "attempts": 2
  }
]
```