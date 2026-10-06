# toolchain

## Purpose

Нормативный контракт эталона: наблюдаемое поведение, ограничения и проверяемые сценарии, обязательные при переносе в новые проекты.

## Requirements

### Requirement: Официальные Go tools
Go-инструменты MUST регистрироваться tool directive через go get -tool с точной версией.
Запуск SHALL использовать go tool и изолированные tools/*.mod; tools.go MUST отсутствовать.

#### Scenario: Чистый clone
- **WHEN** разработчик запускает task tools на Go 1.27.1
- **THEN** Go SHALL загрузить зафиксированные инструменты без глобального go install @latest
