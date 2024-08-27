# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m main template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/main .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Проверка доли покрытия кода тестами

В проекте используются интеграционные тесты, которые конфликтуют с авто-тестами темплейта (metrictests). Для раздельного тестирования, интеграционные тесты используют build tag 'integration'. При запуске логального тестирования и для проверки доли покрытия кода тестами, необходимо указать опцию '-tags=integration', чтобы включить интеграционные тесты.

```bash
go test -v -coverpkg=./... -coverprofile=profile.temp ./... -tags=integration
```

Удаляем сгенерированные файлы mock*.go из профиля, чтобы не влияли на результат вычисления покрытия

```bash
cat profile.temp | grep -v "mock_store.go" > profile.cov

```

```bash
go tool cover -func profile.cov
```
