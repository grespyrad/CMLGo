# Изменение эталона

1. Прочитать AGENTS.md и openspec/specs; создать ветку.
2. Для изменения контракта оформить `openspec/changes/<name>/proposal.md`,
   design.md, tasks.md и delta specs с Requirement/Scenario; проверить `npm run specs:check`.
3. Выполнить тест, который доказывает новое поведение или воспроизводит ошибку.
4. Выполнить `go tool -modfile=tools/task.mod task check`; для сервиса при изменении
   хранения — также integration и db:check/db:lint на отдельной dev-базе.
5. Отправить PR с описанием поведения, результатов проверок и ограничений.

Не менять generated вручную: protobuf → task generate → task generate:check.
Не добавлять обходы хуков, baseline замечаний или массовые nolint.
