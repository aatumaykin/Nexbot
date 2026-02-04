// Спецификация Spawn Tool
//
// SpawnTool предназначен для создания subagents (подагентов) с изолированными сессиями.
//
// ИСПОЛЬЗОВАНИЕ:
//
// 1. Создать обертку для subagent.Manager:
//    spawnFunc := func(ctx context.Context, parentSession string, task string) (string, error) {
//        subagent, err := manager.Spawn(ctx, parentSession, task)
//        if err != nil {
//            return "", err
//        }
//        result := map[string]string{
//            "id":      subagent.ID,
//            "session": subagent.Session,
//        }
//        data, _ := json.Marshal(result)
//        return string(data), nil
//    }
//
// 2. Создать инструмент:
//    spawnTool := tools.NewSpawnTool(spawnFunc)
//
// 3. Зарегистрировать в loop:
//    looper.RegisterTool(spawnTool)
//
// ПАРАМЕТРЫ:
//
// - task (required): Описание задачи для подагента
// - timeout_seconds (optional): Таймаут в секундах (по умолчанию: 300)
//
// ОТВЕТ:
//
// Возвращает JSON с информацией о созданном подагенте:
// {
//   "id": "uuid-subagent-id",
//   "session": "subagent-session-id"
// }
//
// ПРИМЕР ВЫЗОВА:
//
// {
//   "tool": "spawn",
//   "arguments": {
//     "task": "Проанализировать этот документ и написать резюме",
//     "timeout_seconds": 600
//   }
// }
//
// ПРИМЕР ОТВЕТА:
//
// Subagent spawned with ID: {"id":"550e8400-e29b-41d4-a716-446655440000","session":"subagent-1234567890"}
//
// ПРИМЕЧАНИЯ:
//
// - SpawnTool использует интерфейс ContextualTool для поддержки контекста
// - Таймаут применяется к контексту при создании подагента
// - parentSession в текущей реализации всегда "parent", может быть улучшено
// - При ошибке возвращается описательное сообщение об ошибке
