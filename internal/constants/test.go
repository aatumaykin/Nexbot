// Package constants содержит константы для тестов
package constants

import "time"

// TestRequestTimeout - таймаут для тестовых запросов LLM
const TestRequestTimeout = 30 * time.Second

// TestTemperature - температура для тестовых запросов
const TestTemperature = 0.7

// TestMaxTokens - максимальное количество токенов для тестов
const TestMaxTokens = 200

// TestDefaultModel - модель по умолчанию для тестов
const TestDefaultModel = "glm-4.7"

// TestMessage - тестовое сообщение для LLM
const TestMessage = "Hello, world! Please respond with a friendly greeting."

// Сообщения для тестов

// TestMsgProviderInitialized - сообщение о том, что провайдер инициализирован
const TestMsgProviderInitialized = "LLM provider initialized successfully"

// TestMsgSending - сообщение об отправке запроса
const TestMsgSending = "Sending test request to LLM provider"

// TestMsgMessage - сообщение о сообщении
const TestMsgMessage = "Message"

// TestMsgRequestFailed - сообщение о неудачном запросе
const TestMsgRequestFailed = "Request failed"

// TestMsgPossibleCauses - сообщение о возможных причинах
const TestMsgPossibleCauses = "Possible causes:"

// TestMsgCauseAPIKey - сообщение о причине: неверный API ключ
const TestMsgCauseAPIKey = "Invalid API key"

// TestMsgCauseNetwork - сообщение о причине: проблема с сетью
const TestMsgCauseNetwork = "Network connectivity issue"

// TestMsgCauseUnavail - сообщение о причине: сервис недоступен
const TestMsgCauseUnavail = "Service temporarily unavailable"

// TestMsgCauseRateLimit - сообщение о причине: превышен лимит запросов
const TestMsgCauseRateLimit = "Rate limit exceeded"

// TestMsgTroubleshooting - сообщение об устранении проблем
const TestMsgTroubleshooting = "Troubleshooting steps:"

// TestMsgStepAPIKey - шаг: проверка API ключа
const TestMsgStepAPIKey = "1. Verify your API key is correct"

// TestMsgStepConnection - шаг: проверка подключения
const TestMsgStepConnection = "2. Check your network connection"

// TestMsgStepRetry - шаг: повторная попытка
const TestMsgStepRetry = "3. Retry the request after a short delay"

// TestMsgStepStatus - шаг: проверка статуса сервиса
const TestMsgStepStatus = "4. Check the service status page"

// TestMsgSuccess - сообщение об успехе
const TestMsgSuccess = "Test completed successfully"

// TestMsgResponseDetails - сообщение о деталях ответа
const TestMsgResponseDetails = "Response details:"

// TestMsgModel - сообщение о модели
const TestMsgModel = "Model"

// TestMsgLatency - сообщение о задержке
const TestMsgLatency = "Latency"

// TestMsgFinishReason - сообщение о причине завершения
const TestMsgFinishReason = "Finish reason"

// TestMsgResponseContent - сообщение о содержании ответа
const TestMsgResponseContent = "Response content"

// TestMsgContentQuote - кавычки для содержания
const TestMsgContentQuote = `"`

// TestMsgTokenUsage - сообщение о использовании токенов
const TestMsgTokenUsage = "Token usage"

// TestMsgPromptTokens - количество prompt токенов
const TestMsgPromptTokens = "Prompt tokens"

// TestMsgCompTokens - количество completion токенов
const TestMsgCompTokens = "Completion tokens"

// TestMsgTotalTokens - общее количество токенов
const TestMsgTotalTokens = "Total tokens"

// TestMsgToolCalls - сообщение о вызовах инструментов
const TestMsgToolCalls = "Tool calls"

// TestMsgToolCallItem - элемент вызова инструмента
const TestMsgToolCallItem = "Tool call"

// TestMsgStopNormal - сообщение о нормальном завершении
const TestMsgStopNormal = "Stopped normally"

// TestMsgStopLength - сообщение о завершении по длине
const TestMsgStopLength = "Stopped due to max tokens"

// TestMsgStopToolCalls - сообщение о завершении для вызова инструментов
const TestMsgStopToolCalls = "Stopped for tool calls"

// TestMsgStopError - сообщение о завершении из-за ошибки
const TestMsgStopError = "Stopped due to error"

// TestMsgAllPassed - сообщение о прохождении всех тестов
const TestMsgAllPassed = "All tests passed"
