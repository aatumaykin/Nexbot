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

// Сообщения для команды test (с эмодзи)

// TestMsgLoadingConfig - сообщение о загрузке конфигурации
const TestMsgLoadingConfig = "📄 Loading configuration: %s\n"

// TestMsgConfigLoaded - сообщение об успешной загрузке конфигурации
const TestMsgConfigLoaded = "✅ Configuration loaded"

// TestMsgProviderNotSupported - сообщение о неподдерживаемом провайдере
const TestMsgProviderNotSupported = "❌ LLM provider '%s' is not yet supported (only 'zai' is supported)\n"

// TestMsgAPIKeyNotConfigured - сообщение о не настроенном API ключе
const TestMsgAPIKeyNotConfigured = "❌ Z.ai API key is not configured in [llm.zai.api_key]"

// TestMsgFailedToInitLogger - сообщение о неудачной инициализации логгера
const TestMsgFailedToInitLogger = "❌ Failed to initialize logger: %v\n"

// TestMsgInitializingProvider - сообщение об инициализации провайдера
const TestMsgInitializingProvider = "🔌 Initializing Z.ai provider...\n"

// TestMsgProviderInitialized - сообщение об инициализированном провайдере
const TestMsgProviderInitialized = "✅ Z.ai provider initialized (model: %s)\n\n"

// TestMsgSendingRequest - сообщение об отправке тестового запроса
const TestMsgSendingRequest = "📨 Sending test request...\n"

// TestMsgSendingRequestMessage - сообщение о сообщении запроса
const TestMsgSendingRequestMessage = "   Message: %q\n\n"

// TestMsgRequestFailed - сообщение о неудачном запросе
const TestMsgRequestFailed = "\n❌ Request failed: %v\n\n"

// TestMsgPossibleCauses - сообщение о возможных причинах
const TestMsgPossibleCauses = "Possible causes:\n"

// TestMsgCauseAPIKey - сообщение о неверном API ключе
const TestMsgCauseAPIKey = "  • Invalid or expired API key (check ZAI_API_KEY)\n"

// TestMsgCauseNetwork - сообщение о проблемах с сетью
const TestMsgCauseNetwork = "  • Network connectivity issues\n"

// TestMsgCauseUnavail - сообщение о недоступности сервиса
const TestMsgCauseUnavail = "  • Z.ai API is temporarily unavailable\n"

// TestMsgCauseRateLimit - сообщение о превышении лимита
const TestMsgCauseRateLimit = "  • Rate limit exceeded (too many requests)\n"

// TestMsgTroubleshooting - сообщение об устранении проблем
const TestMsgTroubleshooting = "\nTroubleshooting steps:\n"

// TestMsgStepVerifyAPIKey - шаг проверки API ключа
const TestMsgStepVerifyAPIKey = "  1. Verify your API key in config.toml\n"

// TestMsgCheckConnection - шаг проверки подключения
const TestMsgCheckConnection = "  2. Check your internet connection\n"

// TestMsgTryAgain - шаг повторной попытки
const TestMsgTryAgain = "  3. Try again in a few minutes\n"

// TestMsgCheckStatus - шаг проверки статуса сервиса
const TestMsgCheckStatus = "  4. Check Z.ai status page\n"

// TestMsgRequestSuccessful - сообщение об успешном запросе
const TestMsgRequestSuccessful = "✅ Request successful!\n\n"

// TestMsgResponseDetails - сообщение о деталях ответа
const TestMsgResponseDetails = "📥 Response Details:\n"

// TestMsgResponseModel - сообщение о модели ответа
const TestMsgResponseModel = "   Model:        %s\n"

// TestMsgResponseLatency - сообщение о задержке ответа
const TestMsgResponseLatency = "   Latency:      %v\n"

// TestMsgFinishReason - сообщение о причине завершения
const TestMsgFinishReason = "   Finish Reason: %s\n\n"

// TestMsgResponseContent - сообщение о содержании ответа
const TestMsgResponseContent = "📝 Response Content:\n"

// TestMsgResponseContentText - текст содержания ответа
const TestMsgResponseContentText = "   %q\n\n"

// TestMsgTokenUsage - сообщение о использовании токенов
const TestMsgTokenUsage = "📊 Token Usage:\n"

// TestMsgPromptTokens - количество prompt токенов
const TestMsgPromptTokens = "   Prompt Tokens:     %6d\n"

// TestMsgCompletionTokens - количество completion токенов
const TestMsgCompletionTokens = "   Completion Tokens: %6d\n"

// TestMsgTotalTokens - общее количество токенов
const TestMsgTotalTokens = "   Total Tokens:      %6d\n\n"

// TestMsgToolCalls - сообщение о вызовах инструментов
const TestMsgToolCalls = "🔧 Tool Calls: %d\n"

// TestMsgToolCallItem - элемент вызова инструмента
const TestMsgToolCallItem = "   %d. %s(%s)\n"

// TestMsgStopNormal - сообщение о нормальном завершении
const TestMsgStopNormal = "✨ Model completed generation normally"

// TestMsgStopLength - сообщение о завершении по длине
const TestMsgStopLength = "⚠️  Model stopped due to max_tokens limit"

// TestMsgStopToolCalls - сообщение о завершении для вызова инструментов
const TestMsgStopToolCalls = "🔧 Model requested tool/function calls"

// TestMsgStopError - сообщение о завершении из-за ошибки
const TestMsgStopError = "❌ Model stopped due to an error"

// TestMsgAllPassed - сообщение о прохождении всех тестов
const TestMsgAllPassed = "\n✨ All checks passed! Your LLM provider is working correctly."
