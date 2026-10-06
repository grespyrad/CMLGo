# Примеры интеграции сервиса

[service-ru.cml](service-ru.cml) и [service-en.cml](service-en.cml) описывают
одну условную предметную область: приём запросов от внешнего сервиса. Первый
использует русские имена и встроенные DDD aliases; второй — английские имена
и ключевые слова. Описания правил в обоих вариантах на русском.
Имена библиотек и CLI остаются CMLGo и cml-lint.

## Возможности с v0.2.0

| Область Invariant | Английское имя | Правило |
|---|---|---|
| BoundedContext | StableRequestIdentity | Одинаковый requestId обозначает ту же операцию |
| Aggregate | SingleOwner | Состояние запроса изменяет владелец агрегата |
| Entity | NonNegativeAttempts | Число попыток неотрицательно |
| ValueObject | PositiveLimit | Предел попыток положителен |

`rule` описывает ограничение; `rationale` объясняет причину. `testedBy` и
`implementedBy` — условные непрозрачные ссылки с префиксом `example.`.
В репозитории нет указанной реализации сервиса или этих целевых тестов.
Validator проверяет структуру декларации, уникальность и непустые ссылки,
но не исполняет правило и не ищет целевые файлы. Тесты examples проверяют
валидность самих моделей и сохранение данных при обработке.

Инвариант контекста не устанавливает транзакцию между агрегатами.
Command и DomainEvent описывают модель, без runtime доставки сообщений.

## Имена и диалекты

Unicode identifiers и DDD aliases доступны в обычном режиме. `Invariant`,
`rule`, `rationale`, `testedBy`, `implementedBy` остаются английскими:
встроенный русский словарь переводит только поддержанные DDD keywords.
Пользовательский словарь расширяет keywords; имена и строки не переводятся.
`Marshal` возвращает canonical English keywords и сохраняет выбранные имена.
Для русского вывода keywords используйте `MarshalWithDialect` с RussianAliases.
Получить английские имена из русской модели автоматическим Marshal нельзя:
именно поэтому здесь сохранены две отдельные модели.

`--original` / `Dialect{Original: true}` запрещает расширения v0.2.0.
Оба service examples содержат Invariant и ожидаемо отклоняются в этом режиме,
в том числе английский вариант. Для Original используйте условные модели
[testdata/reference/service-before.cml](../testdata/reference/service-before.cml) и
[service.cml](../testdata/reference/service.cml). Они описывают
исходный и целевой контракты интеграции сервиса без расширений Invariant.

## CLI

В cml-lint после сборки `bin/cm`:

```sh
bin/cm validate -i examples/service-ru.cml --json
bin/cm validate -i examples/service-en.cml --json
bin/cm lint -i examples/service-ru.cml
bin/cm format --check examples/service-en.cml
bin/cm generate -g plantuml -i examples/service-en.cml -o bin/example-diagrams
bin/cm --original validate -i testdata/reference/service.cml
```

Библиотечные примеры API находятся в CMLGo `example_test.go` и выполняются
через `go test ./...`. Парные CML файлы одинаковы в обоих репозиториях.

## Модель эталонного Go-сервиса

[etalon-service.cml](etalon-service.cml) сохраняет имена Users, Products, Orders
и Deliveries; ExternalService представляет внешний storage port. [etalon-service-neutral.cml](etalon-service-neutral.cml) использует
Accounts, Resources, Requests и Tasks. Оба варианта описывают тот же контракт.

Источник — [etalon-go-service](https://github.com/grespyrad/etalon-go-service),
проверенный Go snapshot f5e9b4a. Использованы определения из internal/domain,
операции internal/service, consumers и контракты repositories. Конфигурация,
данные развёртывания, примеры пользователей и тестовые значения из эталона
не копируются. Это ручная модель контракта; автоматической генерации из Go нет.

| Go-объект / операция | CML с исходными именами | Обобщённый вариант |
|---|---|---|
| users.User | Users.User | Accounts.Account |
| products.Product | Products.Product | Resources.ResourceRecord |
| orders.Order | Orders.Order | Requests.Request |
| orders.Product | Orders.OrderedProduct | Requests.RequestedResource |
| deliveries.Delivery | Deliveries.Delivery | Tasks.TaskRecord |
| OrderService.HandleCreatedOrders | ProcessOrders | ProcessRequests |
| ProductInited / CheckProduct | ProductInitialization | ResourceInitialization |

Root aggregate и value objects описывают границы данных. Service operations
описывают порты; Application Flow показывает порядок событий и переходов.
CML CREATED/PROCESSING соответствуют Go created/processing, INIT/PUBLISHED/RESERVED —
init/published/reserved. OrderedProduct — проекция продукта в контексте заказа.

В CMLGo пакет examples/reference содержит небольшую Go-реализацию трёх правил
и внешние тесты. Ссылки examples/reference.* в этих двух моделях относятся
к реальным нейтральным methods/tests этой библиотеки, в отличие от условных
example.* в вводных service-ru/service-en examples. Реализация покрывает переходы
состояний и сброс ссылки на вложение; PostgreSQL, Kafka, UUID, HTTP и S3 в неё
не входят. Пример не обещает транзакционность между SQL и Kafka или устранение
дубликатов при создании Delivery: таких гарантий эта модель источника не доказывает.
Синтаксис Invariant — расширение CMLGo; validator не исполняет эти правила.

```sh
bin/cm validate -i examples/etalon-service.cml --json
bin/cm lint -i examples/etalon-service-neutral.cml
bin/cm generate -g plantuml -i examples/etalon-service-neutral.cml -o bin/etalon-diagrams
```

Сигнатуры Repository показывают проекцию портов на CML: context.Context и
callback-параметры исходного Go API опущены. Метод checkProduct допускает
INIT/PUBLISHED/RESERVED -> PUBLISHED: повторная публикация в application service
обрабатывается как успешная проверка; доменный Publish при этом возвращает
ошибку повторной публикации. Это разные уровни контракта. Внешний ImageStorage
представлен как ExternalService, без учётной записи, endpoint и provider settings.

Bounded contexts здесь являются логическими частями одного Go-сервиса. ContextMap
показывает зависимости моделей и портов; отдельные deployment units из него
не следуют. Модель отражает выбранные контракты, а остальные input validations
и инфраструктурные сценарии остаются в исходном Go-эталоне.
