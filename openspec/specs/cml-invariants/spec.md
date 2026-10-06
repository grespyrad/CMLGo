# cml-invariants

## Purpose

Декларативные DDD-инварианты как документированное расширение CMLGo.

## Requirements

### Requirement: Явная область действия
Invariant SHALL иметь имя и обязательный rule STRING. Область SHALL определяться
непосредственным BoundedContext, Aggregate, Entity или ValueObject. rationale,
testedBy и implementedBy SHALL хранить объяснение и opaque ссылки на тесты/код.
Предикаты MUST не исполняться; ссылки MUST не читаться с диска и не запускаться.

#### Scenario: Доменное правило
- **WHEN** агрегат содержит Invariant SingleOwner с rule и двумя связями
- **THEN** Parse, CheckSyntax, Validate и Unmarshal SHALL принимать его и сохранять typed поля.

### Requirement: Корректность декларации
Validator MUST отклонять пустые rule/ссылки, duplicate имена в одной области и
duplicate ссылки. Отсутствие trace links SHALL давать warnings без изменения valid.
Одинаковое имя в разных областях SHALL быть допустимо.

#### Scenario: Незавершённая спецификация
- **WHEN** инвариант имеет непустой rule без testedBy и implementedBy
- **THEN** valid SHALL быть true с warnings об отсутствующих trace links.

### Requirement: Сохранение и визуализация
Typed model и Marshal SHALL сохранять все invariant данные. PlantUML SHALL
показывать имя, область и правило. Оригинальные CML модели SHALL оставаться валидны.
Extension MUST явно отличаться от возможностей официального Context Mapper 6.12.0.

#### Scenario: Round trip
- **WHEN** расширенная модель сериализуется и разбирается снова
- **THEN** rule, rationale и trace links SHALL совпадать.
