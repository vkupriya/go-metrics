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

В проекте используются интеграционные тесты, которые конфликтуют с авто-тестами темплейта (metrictests). Для раздельного тестирования, интеграционные тесты используют build tag 'integration'. При запуске локального тестирования и для проверки доли покрытия кода тестами, необходимо указать опцию '-tags=integration', чтобы включить интеграционные тесты.

```bash
go test -v -coverpkg=./... -coverprofile=profile.temp ./... -tags=integration
```

Удаляем сгенерированные файлы mock*.go из профиля, чтобы не влияли на результат вычисления покрытия

```bash
cat profile.temp | egrep -v "mock_store.go|test/main|staticlint|keygen|proto|.pb.go" > profile.cov

```

```bash
go tool cover -func profile.cov
```

Use html view to observe code lines coverage with testing:

```bash
go tool cover -html=profile.cov -o coverage.html
```

## Проверка кода мультическером

Мультическер (multichecker) проверяет код всеми стандартными анализаторами *golang.org/x/tools/go/analysis/passes* , а также кастомным анализатором, который запрещает использовать os.Exit() в main. Запускается multichecker следующим образом из корня проекта:

```bash
./multichecker -all ../.././...
```

## Проверка кода staticcheck

Файл конфигурации тестов в файле staticcheck.conf, который включает все SA тесты. Запускаются они следующим образом и корня проекта:

```bash
staticcheck ./...
```

## Флаги компилятора

В коде сервера метрик предусмотрены установка и вывод версии (buildVersion), даты сборки (buildDate) и коммит версии (buildCommit). Пример установки флагов при сборке:

```bash
go build -ldflags "-X main.buildVersion=v1.0.1 -X 'main.buildDate=$(date +'%Y/%m/%d')' -X main.buildCommit=cb92c23" -o server
```

При сборке и запуске:
```bash
go run -ldflags "-X main.buildVersion=v1.0.1 -X 'main.buildDate=$(date +'%Y/%m/%d')' -X main.buildCommit=cb92c23" main.go
```