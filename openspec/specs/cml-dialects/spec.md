# cml-dialects

## Purpose

Русские ключевые слова, Unicode имена и задаваемые пользователем алиасы.

## Requirements

### Requirement: Русская запись
Русские aliases для подтверждённых книжными источниками DDD entity introducers SHALL отображаться в
каноническую грамматику; английские и русские ключевые слова SHALL смешиваться.
Unicode identifiers SHALL поддерживать letters, digits после первого символа,
combining marks и underscore. Escaped names SHALL обходить keyword collision.
Строки и комментарии MUST не переводиться. Formatter tokens SHALL сохранять raw.

#### Scenario: Русская модель
- **WHEN** ввод содержит ОграниченныйКонтекст Сервис, Агрегат Запросы и Сущность Запрос
- **THEN** parser SHALL строить обычные BoundedContext/Aggregate/Entity, сохраняя имена.

### Requirement: Другой язык и исходная совместимость
Dialect SHALL принимать alias-to-canonical словарь без глобального изменения
состояния; неверные и конфликтующие aliases MUST давать diagnostics.
Original mode SHALL проверять исходную грамматику 6.12.0 с ASCII именами.
Новые English extension keywords MUST быть soft и не резервировать прежние имена.

#### Scenario: Пользовательский словарь
- **WHEN** ParseWithDialect получает Contexto -> BoundedContext
- **THEN** Contexto ServiceIntegration SHALL разбираться; следующие обычные вызовы SHALL не менять поведение.

#### Scenario: Strict original
- **WHEN** Original dialect проверяет русские ключи, Unicode names или Invariant block
- **THEN** такие расширения MUST отклоняться, а исходные модели SHALL проходить.

### Requirement: Обратная сериализация
MarshalWithDialect SHALL выводить выбранные aliases только для grammar tokens;
имена и значения строк MUST сохраняться. Русский словарь SHALL быть доступен как
независимая копия; чужое изменение карты MUST не влиять на последующие вызовы.

#### Scenario: Русская сериализация
- **WHEN** Go модель сериализуется с RussianAliases и разбирается снова
- **THEN** модельные данные SHALL совпадать, а ключевые слова SHALL быть русскими.
