# Строгий профиль Go

Зафиксированы Go 1.27.1, golangci-lint 2.14.0, govulncheck 1.1.4 и Task 3.54.0.
Обновление выполняется явно с проверкой release notes и повторным `task check`.
Задач по расписанию и GitHub Actions нет.

Основная команда:

```sh
go tool -modfile=tools/task.mod task tools
go tool -modfile=tools/task.mod task check
go tool -modfile=tools/task.mod task hooks
```

`check` проверяет точную версию Go, module graphs и checksum, форматирование,
весь линт-профиль, архитектуру, build, vet, race/shuffle unit-тесты,
достижимые уязвимости, OpenSpec и отрицательные примеры срабатывания линтеров.
В сервисе дополнительно проверяется совпадение protobuf/generated/OpenAPI.
Проверка уязвимостей использует актуальную официальную базу; offline-кэш не доказывает отсутствие новых CVE.

## Почему такой конфиг

`linters.default: all`, `govet.enable-all`, `staticcheck: all`, `gocritic.enable-all`,
`revive.enable-all-rules`, `testifylint.enable-all`. Убраны лимиты вывода ошибок,
стандартные presets исключений и скрытие совпадающих замечаний.
Контролируются потерянные ошибки, wrapping/errors.Is, context/cancellation,
SQL rows/transactions, закрытие ресурсов, безопасность TLS, data races,
проверки enum, mutable globals, сложность, документация и форматирование.

Профиль намеренно согласован: deprecated exhaustruct/gomodguard/wsl заменены
активными преемниками. decorder/funcorder не навязывают бессмысленный порядок
объявлений; noinlineerr/nonamedreturns/ireturn/varnamelen конфликтуют с допустимыми
Go-идиомами, конструкторными интерфейсами и контрактами Go APIs. govet.fieldalignment отключён: его autofix переставляет
embedded поля в противоречии с embeddedstructfieldcheck; память оптимизируется
по benchmark и без изменения публичного layout.
Дублирующие правила revive перечислены с пояснениями рядом с настройками.
`iface.unused` выключен: экспортируемые интерфейсы служат контрактами для клиентов
из других пакетов и моков. Остальные проверки iface включены.
Повторение литералов fmt/slog не требует констант; обычные значения проверяет goconst.

Тестовые fixtures и длинные последовательные integration scenarios имеют узкие
исключения для длины, повторов и числовых данных. Suite не запускает subtests
параллельно, а хранение ролевых контекстов допускается только в двух fixtures.
Внешние SDK структуры используют документированные defaults; собственные
структуры проверяются exhaustruct_v5. Генерированный код исключается только по
стандартному маркеру генератора; рукописный generated/embed.go проверяется.
Ошибки, безопасность и ресурсы в тестах остаются под проверками.
`nolint` обязан назвать правило и объяснить конкретную причину.
Новые исключения требуют обоснования и отрицательного теста, если касаются корректности.

## Инструменты и переиспользование

Используем официальный механизм `tool` в Go module files, `go get -tool` и
`go tool`. Старого `tools.go` с blank imports нет. Файлы `tools/*.mod` и `.sum`
изолируют деревья зависимостей инструментов от runtime и друг от друга.
Для golangci-lint это прямо рекомендованный способ, если выбран `go tool`:
https://golangci-lint.run/docs/welcome/install/local/#install-from-sources.
Официальная документация Go: https://go.dev/doc/modules/managing-dependencies.
Версии фиксируются в require, а checksum — в sum.

```sh
go get -tool -modfile=tools/golangci.mod github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
```

Копируйте `.golangci.yml`, `tools/`, общие `scripts/check-*.sh` и задачи Taskfile.
Module path внутри tool module files замените своим. Архитектурные ограничения
адаптируйте к структуре продукта. Не копируйте service-specific bootstrap и инфраструктуру
в библиотеку. Проверяйте профиль: `task policy` намеренно создаёт пять нарушений
(errcheck, gosec, errorlint, depguard, nolintlint) во временном отдельном Git-репозитории.
Ошибка, которую линтер пропустил, завершает эту проверку неуспешно.
Отдельные три пробы доказывают отказ architecture checker при нарушении domain, service и pkg границ.

OpenSpec — отдельный Node CLI, не Go tool. Закреплён @fission-ai/openspec 1.14.0,
установка `npm ci --ignore-scripts`. Контракты поведения находятся в openspec/specs.
OpenAPI описывает HTTP API; библиотека без HTTP не получает искусственный API.
Нативные Git hooks включаются командой `task hooks`, pre-commit проверяет линт,
pre-push выполняет `check`. Файлы hooks входят в шаблон; core.hooksPath не переносится
при clone и включается явно один раз.

## Совместимость security scanner

govulncheck 1.1.4 закреплён как последний стабильный релиз. Его исходный
x/tools 0.29.0 падает при анализе некоторых конструкций Go 1.27; tools/security.mod
явно фиксирует x/tools 0.50.0 и соответствующий module graph. Проверено на полном
сервисе и библиотеке. Версия scanner не заменяется непроверенным @master.

## Специфика CMLGo

Сохранён полный lint profile эталона. Локальные `nolint` complexity относятся
к интерпретации pinned grammar и отдельным многоветвевым semantic constraints;
корректность контролируют 273 conformance fixtures, round-trip, race и fuzz.
Grammar cache публикуется через sync.Once и после загрузки не изменяется.
Generated model structs имеют optional поля: их отсутствие — часть CML-контракта,
поэтому пять типов, используемых в external construction example, исключены из
требования полного struct literal. Проверки ошибок, ресурсов и безопасности
не отключаются на уровне репозитория.

`govulncheck` не обнаружил достижимых Go-уязвимостей. При установке закреплённого
OpenSpec 1.14.0 `npm audit` сообщает dev-only advisory GHSA-vfj7-8cjw-p6xm в braces
и три зависимых предупреждения. Исправленный braces-релиз не указан в audit;
автоматический downgrade OpenSpec до 0.17.2 не применяется. Это tooling dependency,
она не входит в runtime CMLGo и собранный Go-бинарник. Не передавайте этому
закреплённому OpenSpec CLI недоверенные glob patterns. Отчёт audit можно воспроизвести
командой `npm audit --json`; обновление tooling требует отдельной проверки.

## Проверки текущей поставки

`task check` проверяет lint, architecture, build, vet, race/shuffle, module graph,
govulncheck, OpenSpec и policy probes. Собственные примеры дополнительно
проверяются через scripts/check-examples.py, история — его режимом --history.
