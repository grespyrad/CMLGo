# Покрытие DDD в CMLGo

CML 6.12.0 описывает стратегическую модель: domains/subdomains, bounded contexts,
context map и отношения, ownership, use cases/user stories. Тактическая часть
описывает aggregates/roots, entities/value objects, events/commands, services,
repositories, attributes/references, lifecycle states и transitions. Application
flows и coordination задают связи между командами, событиями и операциями.

Источники: [CML reference](https://contextmapper.org/docs/language-reference/),
[Aggregate](https://contextmapper.org/docs/aggregate/),
[Application/process](https://contextmapper.org/docs/application-and-process-layer/).
Проверена pinned grammar 6.12.0. `required`, `nullable`, `key`, длины и другие
attribute constraints — ограничения полей; `validate` STRING — настройка
Sculptor, не именованный инвариант агрегата с traceability.

## Добавленное расширение CMLGo

```cml
BoundedContext ServiceIntegration {
  Aggregate Requests {
    Invariant SingleOwner {
      rule "Состояние запроса изменяет владелец агрегата"
      rationale "Повторная доставка не меняет владельца состояния"
      testedBy "example.contracts.TestSingleOwner"
      implementedBy "example.requests.Handle"
    }
  }
}
```

Invariant разрешён внутри BoundedContext, Aggregate, Entity и ValueObject.
Область — непосредственный контейнер. Имена уникальны в этой области; rule и
ссылки не могут быть пустыми; duplicate links дают ошибки. Отсутствие тестовых
или implementation links — warning. Ссылки opaque: validator проверяет наличие
декларации, не существование/успех теста. Предикаты не исполняются. Инвариант
контекста не означает атомарную транзакцию между его агрегатами.

Go structs содержат `Invariants []*Invariant`; поля `Name`, `Rule`, `Rationale`,
`TestedBy`, `ImplementedBy`. Marshal/Unmarshal сохраняют их, PlantUML показывает
scope/name/rule. Upstream grammar не меняется: overlay находится в отдельном файле.
Новые English ключи soft и не резервируют ранее допустимые names.

## Границы подробной спецификации

CML + инварианты дают структуру, правила и trace links, но не полную исполняемую
спецификацию системы. Бизнес-сценарии, pre/postconditions операций, reject/error
contracts, transaction/consistency boundaries, idempotency/retry/timeouts,
saga compensation, authorization и acceptance tests требуют явных контрактов.
Flows описывают порядок взаимодействий, не доказывают runtime timing, доставку
или concurrency. `testedBy` следует связать с реальными тестами проекта; эту
работу выполняет реализация, а не parser. Для DDD описывать сначала ubiquitous
language, ownership и invariants, затем command/event contracts и сценарии.

## Языки записи

English остаётся основным выводом Marshal. Подтверждённые Russian aliases
и Unicode names принимаются без перевода данных. `RussianAliases()` возвращает
копию словаря; `Dialect{Aliases: map[string]string{...}}` добавляет другой язык.
`ParseWithDialect`, `CheckSyntaxWithDialect`, `TokenizeWithDialect`,
`ValidateWithDialect`, `ValidateFileWithDialect`, `UnmarshalWithDialect` и
`MarshalWithDialect` принимают его. Aliases — single identifier, canonical target
должен быть известным keyword; конфликт словаря даёт ошибку. Строки не переводятся.
Unicode names не нормализуются: разные codepoint последовательности различаются.
`^` экранирует совпадение имени с keyword. `Dialect{Original:true}` проверяет
исходный ASCII CML 6.12.0 без расширений и aliases.

[Книжный словарь и страницы источников](books/SOURCES.md).

[Две полные модели с русскими и английскими именами](../examples/README.md).
Trace links в этом фрагменте условны.
