# cml-validation

## Purpose

Независимая Go-библиотека и CLI для проверки исходного CML. Контракт основан
на эталоне etalon-go-library и грамматиках Context Mapper 6.12.0.

## Requirements

### Requirement: Полная грамматика CML
Parser SHALL принимать все правила стратегической и тактической грамматики
6.12.0, включая домены, требования, stakeholders, values, application flows,
coordination, services, repositories, resources, traits, DTO, enum и constraints.
Неизвестные конструкции и оставшийся после модели текст MUST отклоняться.

#### Scenario: Учебные модели
- **WHEN** проверяются синтетические модели интеграции сервиса service.cml и service-before.cml
- **THEN** обе модели SHALL быть валидны.

#### Scenario: Синтаксическая ошибка
- **WHEN** пропущена скобка, строка не закрыта или указана неизвестная конструкция
- **THEN** диагностика SHALL содержать файл, строку, колонку и код ошибки.

### Requirement: Семантика и ссылки
Validator SHALL проверять typed references, imports, uniqueness, relationship
roles, map membership, aggregate roots/lifecycle, team ownership, flows,
coordination и ограничения тактических свойств. Warnings MUST не менять valid.
Поведение SHALL сверяться с официальным валидатором, расхождения MUST быть явны.

#### Scenario: Невалидные ссылки и агрегат
- **WHEN** модель содержит отсутствующий context, две aggregateRoot или duplicate context
- **THEN** valid SHALL быть false и ошибки SHALL объяснять нарушенное правило.

#### Scenario: Импорты
- **WHEN** модель импортирует соседний файл, включая повторный или циклический импорт
- **THEN** ссылки SHALL разрешаться без бесконечной рекурсии; отсутствующий файл MUST давать ошибку.

### Requirement: Unicode без изменения формата
Строки и комментарии SHALL принимать UTF-8, включая кириллицу, CJK и emoji.
Original identifiers SHALL следовать исходному CML; расширенный режим SHALL
допускать Unicode names и escaped identifiers.
Невалидный UTF-8 MUST отклоняться; YAML MUST не вводиться. Декларативные расширения описаны отдельно в cml-invariants;
Original dialect MUST сохранять исходный синтаксис.

#### Scenario: Unicode описание
- **WHEN** domainVisionStatement содержит "Сервис 🔧 文" и комментарий содержит Unicode
- **THEN** модель SHALL приниматься без изменения содержимого строки.

### Requirement: Публичный API
Root package SHALL предоставлять Parse, CheckSyntax, Validate, ValidateFile,
Tokenize и AST; типизированные Unmarshal/Marshal SHALL описываться в cml-model.
API MUST не запускать процессы, не требовать JVM и быть безопасным для независимых goroutines.

#### Scenario: Интеграция
- **WHEN** внешняя программа вызывает Validate для UTF-8 CML
- **THEN** Result SHALL содержать valid/diagnostics и исходные позиции.

### Requirement: Проверяемая совместимость
Semantics SHALL сверяться с полным Xtext resource validator 6.12.0; чтение
resource.getErrors без запуска semantic validation MUST не служить oracle семантики.
273 сохранённых fixtures SHALL проверяться локальными Go-тестами без JVM.

#### Scenario: Conformance corpus
- **WHEN** запускается TestOfficialConformance
- **THEN** valid SHALL совпадать с сохранёнными ответами официального validator.
