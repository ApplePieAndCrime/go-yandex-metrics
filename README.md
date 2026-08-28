# go-yandex-metrics

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Бенчмарки и profiling

Добавлены бенчмарки для хранилища метрик и HTTP-обработчика bulk update:

```bash
go test ./internal/repository -bench BenchmarkMemoryStorage -benchmem
go test ./internal/handler/server -bench BenchmarkBulkUpdateMetrics -benchmem
```

Для профилирования использовался реалистичный непустой набор с обновлением counter:

```bash
go test ./internal/repository -bench BenchmarkMemoryStorage -benchmem -benchtime=100x -memprofile profiles/base.pprof
```

### Результаты оптимизации MemoryStorage
В хранилище был добавлен индекс `map[metricKey]int`, где ключ состоит из имени и типа метрики, а значение содержит позицию метрики в срезе. Благодаря этому поиск в `GetMetricsByID` и поиск существующей метрики в `SaveMetrics` выполняются в среднем за O(1) вместо O(n). Возвращаемые метрики глубоко копируются вместе со значениями `Delta` и `Value`, поэтому вызывающий код не получает доступ к изменяемой памяти хранилища и не создаёт гонку данных.

После оптимизации и добавления безопасного копирования профиль `profiles/result.pprof` показывает 46,42 МБ выделенной памяти — примерно на 36% меньше исходного значения. Объём выделений непосредственно в `SaveMetrics` сократился с 61,40 до 15,50 МБ, то есть примерно на 75%. Основная часть оставшихся выделений приходится на создание предварительно заполненных среза и индекса в `NewMemoryStorage`, а также на безопасные копии возвращаемых метрик. Имена метрик теперь подготавливаются до запуска `b.Loop()` и не искажают измеряемую операцию.

Результат контрольного запуска:

```text
BenchmarkMemoryStorageSaveMetrics-10        301952 ns/op   584009 B/op   11012 allocs/op
BenchmarkMemoryStorageGetMetricsByID-10        131.2 ns/op       72 B/op       2 allocs/op
```

## Прогон linter
```bash
go run ./cmd/linter ./...
```

## Сборка с метаданными

Версия, дата сборки и хеш коммита передаются в `buildVersion`, `buildDate` и `buildCommit` с помощью флага `-ldflags`. Например, для сервера:

```bash
go build -ldflags="-X 'main.buildVersion=v1.0.0' -X 'main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'main.buildCommit=$(git rev-parse HEAD)'" -o server ./cmd/server
```

Для агента используется та же команда с другим пакетом и именем бинарного файла:

```bash
go build -ldflags="-X 'main.buildVersion=v1.0.0' -X 'main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'main.buildCommit=$(git rev-parse HEAD)'" -o agent ./cmd/agent
```

Если значение не передано, приложение выведет для него `N/A`.

## Защищённый gRPC-транспорт

Для генерации самоподписанного TLS-сертификата выполните:

```bash
go run ./cmd/gencert \
  -cert server.crt \
  -key server.key \
  -hosts localhost,127.0.0.1
```

Запустите HTTP- и gRPC-серверы, передав сертификат и приватный ключ:

```bash
go run ./cmd/server \
  -g localhost:3200 \
  -grpc-cert server.crt \
  -grpc-key server.key
```

Агенту передаётся тот же сертификат как доверенный:

```bash
go run ./cmd/agent \
  -g localhost:3200 \
  -grpc-cert server.crt
```

Если адрес подключения не совпадает с DNS-именем из сертификата, ожидаемое имя можно задать флагом `-grpc-server-name`. Аналогичные параметры доступны через `GRPC_ADDRESS`, `GRPC_CERT_FILE`, `GRPC_KEY_FILE` (сервер), `GRPC_SERVER_NAME` (агент) и JSON-поля `grpc_address`, `grpc_cert_file`, `grpc_key_file`, `grpc_server_name`.

## Proto
Код из `metrics.proto` генерируется с использованием Opaque API:

```bash
make proto-generation
```
