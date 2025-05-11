# Шина событий

## Описание
Сервис реализует паттерн Publisher-Subscriber (Pub/Sub) на Go с использованием gRPC.
Основные функции:
- Подписка на события по ключу (Subscribe).
- Публикация событий для всех подписчиков ключа (Publish).

## Установка и запуск

### 1. Клонирование репозитория

```bash
git clone https://github.com/emilien-gus/pubsub
cd pubsub
```

### 2. Запуск проекта

```bash
go build -o pubsub cmd/event-bus/main.go && ./pubsub
```

### 3. Запуск тестов

Запуск unit-тестов для модуля pubsub:
```bash
go test -v ./...
```

### 4. Запуск клиентов

Нужно установить [grpcurl] https://github.com/fullstorydev/grpcurl.
Пример создания подписки:
```bash
grpcurl -plaintext -proto proto/pubsub.proto   -d '{"key": "test"}'   localhost:50051 pubsub.PubSub/Subscribe
```
Пример создания публикации:
```bash
 grpcurl -plaintext -proto proto/pubsub.proto   -d '{"key": "test1", "data": "Hello"}'   localhost:50051 pubsub.PubSub/Publish
```
