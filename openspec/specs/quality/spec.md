# quality

## Purpose

Нормативный контракт эталона: наблюдаемое поведение, ограничения и проверяемые сценарии, обязательные при переносе в новые проекты.

## Requirements

### Requirement: Воспроизводимая проверка качества
Репозиторий MUST фиксировать версии Go, линтера, security scanner, Task и OpenSpec.
`task check` SHALL завершаться ошибкой при lint, build, vet, test, module или spec нарушении.

#### Scenario: Нарушение безопасности
- **WHEN** отрицательный пример отключает TLS-проверку сертификата
- **THEN** gosec SHALL обнаружить ошибку, а policy probe SHALL подтвердить её обнаружение

#### Scenario: Проверка проекта
- **WHEN** разработчик запускает task check
- **THEN** проверки SHALL работать без GitHub Actions и внешней Kafka
