# CMLGo

Go-библиотека для Context Mapper Language 6.12.0: parser, типизированные
Go-объекты, сериализация исходного CML и семантическая валидация. Runtime использует
только standard library. Формат CML сохранён; YAML не добавлен. С v0.2.0 доступны декларативные
инварианты, Unicode identifiers и необязательные языковые aliases.

Отдельный [cml-lint](https://github.com/grespyrad/cml-lint) импортирует библиотеку и
предоставляет линтер, форматтер и PlantUML CLI. Оба проекта созданы на основе
[etalon-go-library](https://github.com/grespyrad/etalon-go-library).

```go
result := cml.Validate("model.cml", source)
if !result.Valid {
    // Обработайте result.Diagnostics: файл, позиция, severity и код.
    return cml.ErrInvalidModel
}
var model cml.ContextMappingModel
if err := cml.Unmarshal(source, &model); err != nil {
    return err
}
encoded, err := cml.Marshal(&model)
if err != nil {
    return err
}
// Используйте encoded как CML с canonical keywords.
```

Импорт: `cml "github.com/grespyrad/CMLGo"`. Полные выполняемые примеры API:
[example_test.go](example_test.go). [Парные CML примеры](examples/README.md)
демонстрируют новые инварианты во всех четырёх областях, Unicode names
и русские aliases на модели условного ExternalService.

Все 143 правила pinned Xtext-грамматики имеют Go-представление: structs для
объектов/union, string enums и `Ref[T]` для cross references. Например,
`Aggregate.DomainObjects` содержит `SimpleDomainObject` с вариантами `Entity`,
`ValueObject`, `DomainEvent`, `CommandEvent`, `Enum` и другими. `Ref[T].Name`
сохраняет имя ссылки; `Validate`/`ValidateFile` проверяют её разрешимость.
`Position` и `Raw` — метаданные; обычная JSON-сериализация их исключает.

`Parse(file, source)` строит универсальный AST; `CheckSyntax` выполняет только
проверку грамматики. `Unmarshal` напрямую заполняет типизированную модель.
`Validate` проверяет standalone source в памяти; для imports используйте
`ValidateFile(path)`. Циклические и повторные imports обрабатываются однократно.
Строки и комментарии поддерживают UTF-8, а строки — Unicode escapes и surrogate pairs; identifiers остаются совместимыми с CML
(Unicode расширение с v0.2.0, включая `^escaped`; для ASCII CML 6.12.0
используйте `Dialect{Original: true}`).

`Marshal` сохраняет модельные данные, но не исходные комментарии и layout.
Для форматирования исходного файла с сохранением комментариев используйте cml-lint.
`PlantUML` создаёт context map и class diagrams в памяти; layout и filenames
отличаются от Java generator. SVG/PNG renderer в библиотеку не входит.

## Совместимость и проверки

Сохранены 55 официальных примеров, две условные reference models и 273 фрагмента
из upstream tests. 273 решений по semantic validity совпадают с полным Xtext
validator 6.12.0; corpus round-trip проверяет данные типизированных моделей.
Эти результаты доказывают совместимость на сохранённом corpus, а не на любом
теоретически возможном файле. Диагностики имеют собственные стабильные коды.
Ресурсные пределы: 8 MiB на source, 256 imports, глубина parser 256.

```sh
go test ./...
go tool -modfile=tools/task.mod task tools
go tool -modfile=tools/task.mod task check
go tool -modfile=tools/task.mod task hooks
```

Go 1.27.1; Task 3.54.0, golangci-lint 2.14.0, govulncheck 1.1.4 и OpenSpec 1.14.0
закреплены как в эталоне. Нет GitHub Actions и расписаний. `task check` включает
полный lint profile, race/shuffle, module graph, vulnerability scan, architecture,
negative policy probes и OpenSpec. Сравнение Go/Rust/C++ находится в cml-lint.

Установка: `go get github.com/grespyrad/CMLGo@v0.2.3`.
Локальный `replace` не используется.

Проект основан на Apache-2.0 исходниках [Context Mapper](https://github.com/ContextMapper/context-mapper-dsl).
Attribution и pinned commits: [NOTICE](NOTICE). Это независимая реализация.

Подробнее: [DDD покрытие, инварианты и диалекты](docs/DDD-COVERAGE.md),
[книжные источники русских терминов](docs/books/SOURCES.md). English остаётся
основным выводом Marshal; русский не навязывается.

[Модель эталонного Go-сервиса](examples/README.md#модель-эталонного-go-сервиса):
контексты, агрегаты, repositories, состояния, операции и application flows;
варианты с исходными и обобщёнными именами. Выбранные правила проверяются
нейтральной Go-реализацией в CMLGo examples/reference.
