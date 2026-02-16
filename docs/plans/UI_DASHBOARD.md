# Nexbot UI Dashboard — План развития

**Версия:** 1.0
**Дата:** 2026-02-16
**Источники вдохновения:** PocketPaw, Moltis

---

## Обзор

### Цель
Добавить Web UI dashboard с возможностью:
- Мониторинга и управления агентом из браузера
- Продолжения Telegram чатов из UI
- Создания отдельных UI сессий
- Real-time streaming ответов

### Стек технологий
| Компонент     | Выбор                    | Причина                          |
|---------------|--------------------------|----------------------------------|
| HTTP Router   | chi                      | Легковесный, Go-idiomatic        |
| WebSocket     | gorilla/websocket        | Проверенный, стабильный          |
| Templates     | html/template            | Стандарт Go                      |
| CSS           | Tailwind CDN             | Zero-config, как в PocketPaw     |
| JS            | Alpine.js                | Minimal reactivity               |
| Icons         | Lucide Icons CDN         | Как в PocketPaw                  |

### Архитектура

```
┌─────────────────────────────────────────────────────────────────┐
│                           Gateway Server                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   REST API  │  │  WebSocket  │  │       Static Files      │  │
│  │  /api/*     │  │    /ws      │  │   /static/{css,js}      │  │
│  └──────┬──────┘  └──────┬──────┘  └────────────┬────────────┘  │
│         │                │                      │                │
│         └────────────────┼──────────────────────┘                │
│                          │                                       │
│                   ┌──────▼──────┐                                │
│                   │    Bus      │◀──── WebSocketAdapter          │
│                   │ (extended)  │                                │
│                   └──────┬──────┘                                │
└──────────────────────────┼──────────────────────────────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
    ┌────▼────┐      ┌─────▼─────┐     ┌─────▼─────┐
    │ Telegram│      │   Agent   │     │   Tools   │
    │ Adapter │      │   Loop    │     │ Registry  │
    └─────────┘      └───────────┘     └───────────┘
```

---

## Phase 0: Architecture Refactoring

**Длительность:** 1 неделя
**Приоритет:** P0 (критический)
**Зависимости:** нет

### 0.1 SystemEvent Bus Extension

**Файлы:**
```
internal/bus/
  system_event.go
  system_event_test.go
```

**Задачи:**

- [ ] Определить `SystemEventType` enum:
  ```go
  type SystemEventType string

  const (
      SystemEventToolStart   SystemEventType = "tool_start"
      SystemEventToolEnd     SystemEventType = "tool_end"
      SystemEventToolError   SystemEventType = "tool_error"
      SystemEventThinking    SystemEventType = "thinking"
      SystemEventStreamStart SystemEventType = "stream_start"
      SystemEventStreamChunk SystemEventType = "stream_chunk"
      SystemEventStreamEnd   SystemEventType = "stream_end"
      SystemEventAgentStart  SystemEventType = "agent_start"
      SystemEventAgentEnd    SystemEventType = "agent_end"
      SystemEventError       SystemEventType = "error"
  )
  ```

- [ ] Создать `SystemEvent` struct:
  ```go
  type SystemEvent struct {
      Type      SystemEventType `json:"type"`
      SessionID string          `json:"session_id,omitempty"`
      Data      map[string]any  `json:"data,omitempty"`
      Timestamp time.Time       `json:"timestamp"`
  }
  ```

- [ ] Расширить `MessageBus`:
  ```go
  // Новые поля
  systemCh       chan SystemEvent
  systemSubs     []chan SystemEvent
  systemSubsLock sync.RWMutex

  // Новые методы
  func (b *MessageBus) SubscribeSystem() <-chan SystemEvent
  func (b *MessageBus) UnsubscribeSystem(ch <-chan SystemEvent)
  func (b *MessageBus) PublishSystem(event SystemEvent)
  ```

- [ ] Добавить constructor:
  ```go
  func NewSystemEvent(eventType SystemEventType, sessionID string, data map[string]any) *SystemEvent
  ```

**Тесты:**
- [ ] TestSubscribeSystem
- [ ] TestPublishSystem
- [ ] TestUnsubscribeSystem
- [ ] TestSystemEventJSON

---

### 0.2 OutboundMessage Streaming Extension

**Файлы:**
```
internal/bus/
  events.go           (изменить)
  events_test.go      (изменить)
```

**Задачи:**

- [ ] Добавить поля streaming в `OutboundMessage`:
  ```go
  type OutboundMessage struct {
      // ... existing fields ...

      // Streaming support
      IsStreamChunk bool `json:"is_stream_chunk,omitempty"`
      IsStreamEnd   bool `json:"is_stream_end,omitempty"`
  }
  ```

- [ ] Добавить constructors:
  ```go
  func NewStreamChunkMessage(channelType ChannelType, sessionID, content string) *OutboundMessage
  func NewStreamEndMessage(channelType ChannelType, sessionID string) *OutboundMessage
  ```

**Тесты:**
- [ ] TestNewStreamChunkMessage
- [ ] TestNewStreamEndMessage
- [ ] TestStreamMessageJSON

---

### 0.3 ChannelAdapter Protocol

**Файлы:**
```
internal/channels/
  adapter.go
  adapter_test.go
```

**Задачи:**

- [ ] Определить `Adapter` interface:
  ```go
  // Adapter defines the interface for channel adapters
  type Adapter interface {
      // ChannelType returns the channel type this adapter handles
      ChannelType() bus.ChannelType

      // Start initializes the adapter and subscribes to the message bus
      Start(ctx context.Context, bus *bus.MessageBus) error

      // Stop gracefully shuts down the adapter
      Stop() error

      // Send delivers a message through this channel
      Send(msg bus.OutboundMessage) error
  }
  ```

- [ ] Создать `BaseAdapter`:
  ```go
  // BaseAdapter provides common functionality for channel adapters
  type BaseAdapter struct {
      channelType bus.ChannelType
      bus         *bus.MessageBus
      ctx         context.Context
      cancel      context.CancelFunc
      running     bool
      mu          sync.RWMutex
  }

  func (a *BaseAdapter) Start(bus *bus.MessageBus) error
  func (a *BaseAdapter) Stop() error
  func (a *BaseAdapter) PublishInbound(msg bus.InboundMessage)
  func (a *BaseAdapter) IsRunning() bool
  ```

- [ ] Рефакторинг `TelegramConnector`:
  - [ ] Implement `Adapter` interface
  - [ ] Use `BaseAdapter` embedded struct
  - [ ] Subscribe to outbound via `bus.SubscribeOutbound()`

**Тесты:**
- [ ] TestBaseAdapterStart
- [ ] TestBaseAdapterStop
- [ ] TestBaseAdapterPublishInbound
- [ ] TestAdapterInterface

---

### 0.4 Broadcast Support

**Файлы:**
```
internal/bus/
  broadcast.go
  broadcast_test.go
```

**Задачи:**

- [ ] Добавить `BroadcastOutbound`:
  ```go
  // BroadcastOutbound sends a message to all channel subscribers except excluded
  func (b *MessageBus) BroadcastOutbound(msg OutboundMessage, exclude ChannelType) error
  ```

- [ ] Добавить `BroadcastToChannels`:
  ```go
  // BroadcastToChannels sends a message to specific channels
  func (b *MessageBus) BroadcastToChannels(msg OutboundMessage, channels []ChannelType) error
  ```

**Тесты:**
- [ ] TestBroadcastOutbound
- [ ] TestBroadcastToChannels
- [ ] TestBroadcastExclude

---

## Phase 1: HTTP/WebSocket Gateway

**Длительность:** 1-2 недели
**Приоритет:** P0 (критический)
**Зависимости:** Phase 0

### 1.1 Gateway Configuration

**Файлы:**
```
internal/gateway/
  config.go
  config_test.go
```

**Задачи:**

- [ ] Определить конфигурацию:
  ```go
  type Config struct {
      Enabled    bool   `toml:"enabled"`
      Host       string `toml:"host"`
      Port       int    `toml:"port"`
      TLSEnabled bool   `toml:"tls_enabled"`
      TLSCert    string `toml:"tls_cert"`
      TLSKey     string `toml:"tls_key"`
      Origins    []string `toml:"origins"`  // CORS / WS origins
  }

  func DefaultConfig() Config
  func (c *Config) Validate() error
  ```

- [ ] Добавить в основной config:
  ```toml
  [gateway]
  enabled = true
  host = "127.0.0.1"
  port = 8080
  tls_enabled = false
  origins = ["http://localhost:8080"]
  ```

---

### 1.2 Gateway Server

**Файлы:**
```
internal/gateway/
  server.go
  server_test.go
  routes.go
  middleware.go
```

**Задачи:**

- [ ] Создать `Server` struct:
  ```go
  type Server struct {
      config     Config
      bus        *bus.MessageBus
      httpServer *http.Server
      router     *chi.Mux
      wsHandler  *WebSocketHandler
      templates  *template.Template
      running    bool
      mu         sync.RWMutex
  }

  func NewServer(config Config, bus *bus.MessageBus) (*Server, error)
  func (s *Server) Start(ctx context.Context) error
  func (s *Server) Stop() error
  func (s *Server) Addr() string
  ```

- [ ] Настроить routes:
  ```go
  func (s *Server) setupRoutes() {
      // Static files
      s.router.Get("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

      // Pages
      s.router.Get("/", s.handleIndex)
      s.router.Get("/chat", s.handleChat)
      s.router.Get("/settings", s.handleSettings)

      // WebSocket
      s.router.Get("/ws", s.wsHandler.HandleWebSocket)

      // API
      s.router.Route("/api", func(r chi.Router) {
          r.Get("/health", s.handleHealth)
          r.Get("/channels/status", s.handleChannelsStatus)
          r.Post("/channels/{channel}", s.handleChannelSave)
          r.Post("/channels/{channel}/toggle", s.handleChannelToggle)
          r.Get("/sessions", s.handleSessionsList)
          r.Get("/sessions/{id}", s.handleSessionGet)
      })
  }
  ```

- [ ] Middleware:
  ```go
  func (s *Server) loggingMiddleware(next http.Handler) http.Handler
  func (s *Server) corsMiddleware(next http.Handler) http.Handler
  func (s *Server) recoverMiddleware(next http.Handler) http.Handler
  ```

**Тесты:**
- [ ] TestServerStart
- [ ] TestServerStop
- [ ] TestRoutes
- [ ] TestMiddleware

---

### 1.3 WebSocket Handler

**Файлы:**
```
internal/gateway/
  websocket.go
  websocket_test.go
  connection.go
  connection_test.go
```

**Задачи:**

- [ ] Создать `Connection` struct:
  ```go
  type Connection struct {
      ID         string
      SessionID  string
      conn       *websocket.Conn
      sendCh     chan []byte
      doneCh     chan struct{}
      lastActive time.Time
  }

  func NewConnection(conn *websocket.Conn, sessionID string) *Connection
  func (c *Connection) Send(msg any) error
  func (c *Connection) Close()
  func (c *Connection) ReadLoop(handler func(msg ClientMessage))
  func (c *Connection) WriteLoop()
  ```

- [ ] Создать `WebSocketHandler`:
  ```go
  type WebSocketHandler struct {
      bus         *bus.MessageBus
      adapter     *WebSocketAdapter
      connections sync.Map // sessionID -> *Connection
      upgrader    websocket.Upgrader
  }

  func NewWebSocketHandler(bus *bus.MessageBus) *WebSocketHandler
  func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request)
  func (h *WebSocketHandler) Broadcast(msg any)
  func (h *WebSocketHandler) SendTo(sessionID string, msg any) error
  ```

- [ ] Message protocol:
  ```go
  // Client -> Server
  type ClientMessage struct {
      Action  string         `json:"action"` // "chat", "command"
      Content string         `json:"content,omitempty"`
      Meta    map[string]any `json:"meta,omitempty"`
  }

  // Server -> Client
  type ServerMessage struct {
      Type string `json:"type"` // "message", "system_event", "stream_start", "stream_chunk", "stream_end", "error"
      Data any    `json:"data,omitempty"`
  }
  ```

- [ ] Origin validation (CSWSH protection):
  ```go
  func (h *WebSocketHandler) checkOrigin(r *http.Request) bool {
      origin := r.Header.Get("Origin")
      // Validate against allowed origins
  }
  ```

**Тесты:**
- [ ] TestConnectionSend
- [ ] TestConnectionClose
- [ ] TestWebSocketUpgrade
- [ ] TestOriginValidation
- [ ] TestMessageProtocol

---

### 1.4 WebSocket Channel Adapter

**Файлы:**
```
internal/gateway/
  adapter.go
  adapter_test.go
```

**Задачи:**

- [ ] Создать `WebSocketAdapter`:
  ```go
  // WebSocketAdapter implements channels.Adapter for WebSocket connections
  type WebSocketAdapter struct {
      channels.BaseAdapter
      handler *WebSocketHandler
  }

  func NewWebSocketAdapter(handler *WebSocketHandler) *WebSocketAdapter
  func (a *WebSocketAdapter) Start(bus *bus.MessageBus) error
  func (a *WebSocketAdapter) Send(msg bus.OutboundMessage) error
  ```

- [ ] Subscribe to SystemEvents:
  ```go
  func (a *WebSocketAdapter) Start(bus *bus.MessageBus) error {
      if err := a.BaseAdapter.Start(bus); err != nil {
          return err
      }

      // Subscribe to system events
      a.systemCh = bus.SubscribeSystem()
      go a.systemEventLoop()

      return nil
  }

  func (a *WebSocketAdapter) systemEventLoop() {
      for event := range a.systemCh {
          a.handler.Broadcast(ServerMessage{
              Type: "system_event",
              Data: event,
          })
      }
  }
  ```

- [ ] Handle streaming:
  ```go
  func (a *WebSocketAdapter) Send(msg bus.OutboundMessage) error {
      serverMsg := ServerMessage{Data: msg}

      if msg.IsStreamEnd {
          serverMsg.Type = "stream_end"
      } else if msg.IsStreamChunk {
          serverMsg.Type = "stream_chunk"
      } else {
          serverMsg.Type = "message"
      }

      return a.handler.SendTo(msg.SessionID, serverMsg)
  }
  ```

**Тесты:**
- [ ] TestWebSocketAdapterStart
- [ ] TestWebSocketAdapterSend
- [ ] TestSystemEventBroadcast
- [ ] TestStreamingMessages

---

### 1.5 REST API Endpoints

**Файлы:**
```
internal/gateway/
  api/
    channels.go
    channels_test.go
    sessions.go
    sessions_test.go
    settings.go
    settings_test.go
```

**Задачи:**

- [ ] Channels API:
  ```go
  // GET /api/channels/status
  func (s *Server) handleChannelsStatus(w http.ResponseWriter, r *http.Request)

  // Response:
  // {
  //   "channels": {
  //     "telegram": {"configured": true, "running": true, "error": null},
  //     "websocket": {"configured": true, "running": true, "error": null}
  //   }
  // }

  // POST /api/channels/telegram
  type SaveChannelRequest struct {
      Token string `json:"token"`
  }
  func (s *Server) handleChannelSave(w http.ResponseWriter, r *http.Request)

  // POST /api/channels/telegram/toggle
  type ToggleChannelRequest struct {
      Enabled bool `json:"enabled"`
  }
  func (s *Server) handleChannelToggle(w http.ResponseWriter, r *http.Request)
  ```

- [ ] Sessions API:
  ```go
  // GET /api/sessions
  func (s *Server) handleSessionsList(w http.ResponseWriter, r *http.Request)

  // Response:
  // {
  //   "sessions": [
  //     {
  //       "id": "telegram:123456",
  //       "title": "Discussion about...",
  //       "source": "telegram",
  //       "created_at": "2026-02-16T10:00:00Z",
  //       "updated_at": "2026-02-16T12:30:00Z",
  //       "message_count": 42
  //     }
  //   ]
  // }

  // GET /api/sessions/{id}
  func (s *Server) handleSessionGet(w http.ResponseWriter, r *http.Request)
  ```

- [ ] Health check:
  ```go
  // GET /api/health
  func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request)

  // Response:
  // {
  //   "status": "ok",
  //   "version": "1.0.0",
  //   "uptime": "2h30m",
  //   "channels": {"telegram": "running", "websocket": "running"}
  // }
  ```

**Тесты:**
- [ ] TestChannelsStatus
- [ ] TestChannelSave
- [ ] TestChannelToggle
- [ ] TestSessionsList
- [ ] TestSessionGet
- [ ] TestHealthCheck

---

## Phase 2: Frontend

**Длительность:** 1-2 недели
**Приоритет:** P1 (высокий)
**Зависимости:** Phase 1

### 2.1 Base Templates

**Файлы:**
```
internal/gateway/
  assets/
    templates/
      base.html
      components/
        sidebar.html
        chat.html
        activity.html
        settings.html
```

**Задачи:**

- [ ] `base.html`:
  ```html
  <!DOCTYPE html>
  <html lang="en" x-data="{ theme: localStorage.getItem('theme') || 'dark' }"
        :class="{ 'dark': theme === 'dark' }">
  <head>
      <meta charset="UTF-8">
      <meta name="viewport" content="width=device-width, initial-scale=1.0">
      <title>Nexbot</title>

      <!-- Tailwind CSS -->
      <script src="https://cdn.tailwindcss.com"></script>

      <!-- Alpine.js -->
      <script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>

      <!-- Lucide Icons -->
      <script src="https://unpkg.com/lucide@latest"></script>

      <!-- Custom styles -->
      <link rel="stylesheet" href="/static/css/styles.css">
  </head>
  <body class="bg-gray-900 text-gray-100 h-screen flex overflow-hidden">
      {{ template "sidebar" . }}
      <main class="flex-1 flex flex-col">
          {{ template "content" . }}
      </main>
      <script src="/static/js/app.js"></script>
  </body>
  </html>
  ```

- [ ] `sidebar.html`:
  - Sessions list
  - New Chat button
  - Search input
  - Settings link

- [ ] `chat.html`:
  - Messages container
  - Message input
  - Send button
  - Typing indicator

- [ ] `activity.html`:
  - Tool calls list
  - Thinking indicator
  - Error display

---

### 2.2 JavaScript Application

**Файлы:**
```
internal/gateway/
  assets/
    static/
      js/
        app.js
        websocket.js
        state.js
        chat.js
        sessions.js
```

**Задачи:**

- [ ] `state.js` - Alpine.js reactive state:
  ```javascript
  document.addEventListener('alpine:init', () => {
      Alpine.store('app', {
          // State
          connected: false,
          currentSessionId: null,
          sessions: [],
          messages: {},
          activities: [],
          settings: {},

          // Actions
          connect() { ... },
          disconnect() { ... },
          sendMessage(content) { ... },
          loadSessions() { ... },
          selectSession(id) { ... },
      })
  })
  ```

- [ ] `websocket.js`:
  ```javascript
  class WSClient {
      constructor(url) {
          this.url = url
          this.ws = null
          this.reconnectDelay = 1000
          this.listeners = new Map()
      }

      connect() {
          this.ws = new WebSocket(this.url)
          this.ws.onopen = () => this.onOpen()
          this.ws.onclose = () => this.onClose()
          this.ws.onmessage = (e) => this.onMessage(e)
          this.ws.onerror = (e) => this.onError(e)
      }

      send(data) {
          if (this.ws?.readyState === WebSocket.OPEN) {
              this.ws.send(JSON.stringify(data))
          }
      }

      on(type, callback) {
          this.listeners.set(type, callback)
      }

      reconnect() {
          setTimeout(() => this.connect(), this.reconnectDelay)
          this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000)
      }
  }
  ```

- [ ] `chat.js`:
  ```javascript
  function ChatComponent() {
      return {
          input: '',
          sending: false,
          streaming: false,

          async send() {
              if (!this.input.trim() || this.sending) return

              this.sending = true
              const content = this.input
              this.input = ''

              // Add user message to UI immediately
              this.addMessage({ role: 'user', content })

              // Send to server
              ws.send({ action: 'chat', content })

              this.sending = false
          },

          addMessage(msg) {
              Alpine.store('app').messages.push({
                  id: Date.now(),
                  ...msg,
                  timestamp: new Date().toISOString()
              })
              this.scrollToBottom()
          },

          handleStreamChunk(data) {
              // Update last assistant message
              const messages = Alpine.store('app').messages
              const last = messages[messages.length - 1]
              if (last?.role === 'assistant' && last.streaming) {
                  last.content += data.content
              } else {
                  this.addMessage({
                      role: 'assistant',
                      content: data.content,
                      streaming: true
                  })
              }
          },

          handleStreamEnd() {
              const messages = Alpine.store('app').messages
              const last = messages[messages.length - 1]
              if (last) last.streaming = false
          },

          scrollToBottom() {
              this.$nextTick(() => {
                  const container = document.getElementById('messages')
                  container.scrollTop = container.scrollHeight
              })
          }
      }
  }
  ```

- [ ] `sessions.js`:
  ```javascript
  function SessionsComponent() {
      return {
          search: '',
          loading: false,

          async load() {
              this.loading = true
              const res = await fetch('/api/sessions')
              const data = await res.json()
              Alpine.store('app').sessions = data.sessions
              this.loading = false
          },

          get filtered() {
              if (!this.search) return this.sessions
              const q = this.search.toLowerCase()
              return this.sessions.filter(s =>
                  s.title?.toLowerCase().includes(q) ||
                  s.id.toLowerCase().includes(q)
              )
          },

          select(id) {
              Alpine.store('app').currentSessionId = id
              this.loadMessages(id)
          },

          async loadMessages(sessionId) {
              const res = await fetch(`/api/sessions/${sessionId}`)
              const data = await res.json()
              Alpine.store('app').messages = data.messages || []
          },

          create() {
              Alpine.store('app').currentSessionId = null
              Alpine.store('app').messages = []
          }
      }
  }
  ```

---

### 2.3 CSS Styles

**Файлы:**
```
internal/gateway/
  assets/
    static/
      css/
        styles.css
```

**Задачи:**

- [ ] Base styles:
  ```css
  /* Dark theme variables */
  :root {
      --bg-primary: #111827;
      --bg-secondary: #1f2937;
      --bg-tertiary: #374151;
      --text-primary: #f9fafb;
      --text-secondary: #9ca3af;
      --accent: #3b82f6;
      --accent-hover: #2563eb;
      --border: #374151;
      --success: #10b981;
      --error: #ef4444;
      --warning: #f59e0b;
  }

  /* Scrollbar */
  ::-webkit-scrollbar {
      width: 6px;
  }
  ::-webkit-scrollbar-track {
      background: var(--bg-secondary);
  }
  ::-webkit-scrollbar-thumb {
      background: var(--bg-tertiary);
      border-radius: 3px;
  }

  /* Message formatting */
  .message-content {
      line-height: 1.6;
  }
  .message-content pre {
      background: var(--bg-tertiary);
      padding: 1rem;
      border-radius: 0.5rem;
      overflow-x: auto;
  }
  .message-content code {
      background: var(--bg-tertiary);
      padding: 0.125rem 0.25rem;
      border-radius: 0.25rem;
      font-size: 0.875rem;
  }

  /* Activity panel */
  .activity-item {
      animation: fadeIn 0.3s ease;
  }
  @keyframes fadeIn {
      from { opacity: 0; transform: translateY(-10px); }
      to { opacity: 1; transform: translateY(0); }
  }
  ```

---

### 2.4 UI Components

**Задачи:**

- [ ] **Source Indicator** - показывает источник сообщения:
  ```html
  <span class="source-badge" :class="msg.source">
      <i data-lucide="smartphone" x-show="msg.source === 'telegram'"></i>
      <i data-lucide="monitor" x-show="msg.source === 'web'"></i>
      <span x-text="msg.source"></span>
  </span>
  ```

- [ ] **Typing Indicator**:
  ```html
  <div class="typing-indicator" x-show="streaming">
      <span></span><span></span><span></span>
  </div>
  ```

- [ ] **Activity Panel** - показывает tool calls:
  ```html
  <div class="activity-panel" x-show="activities.length > 0">
      <template x-for="activity in activities" :key="activity.id">
          <div class="activity-item">
              <i :data-lucide="activity.icon"></i>
              <span x-text="activity.name"></span>
              <span x-text="activity.status"></span>
          </div>
      </template>
  </div>
  ```

- [ ] **Connection Status**:
  ```html
  <div class="connection-status" :class="{ 'connected': connected, 'disconnected': !connected }">
      <span class="dot"></span>
      <span x-text="connected ? 'Connected' : 'Disconnected'"></span>
  </div>
  ```

---

## Phase 3: Telegram + UI Bridge

**Длительность:** 1 неделя
**Приоритет:** P1 (высокий)
**Зависимости:** Phase 1, Phase 2

### 3.1 Unified Session Manager

**Файлы:**
```
internal/gateway/
  session_manager.go
  session_manager_test.go
```

**Задачи:**

- [ ] Создать `SessionManager`:
  ```go
  type SessionManager struct {
      store     SessionStore
      aliases   map[string]string // session_key -> canonical_key
      mu        sync.RWMutex
  }

  type Session struct {
      ID           string    `json:"id"`
      CanonicalKey string    `json:"canonical_key"`
      Source       string    `json:"source"` // "telegram", "web"
      Title        string    `json:"title"`
      CreatedAt    time.Time `json:"created_at"`
      UpdatedAt    time.Time `json:"updated_at"`
      MessageCount int       `json:"message_count"`
  }

  func NewSessionManager(store SessionStore) *SessionManager
  func (m *SessionManager) GetOrCreate(key string, source string) (*Session, error)
  func (m *SessionManager) Get(key string) (*Session, error)
  func (m *SessionManager) List() ([]*Session, error)
  func (m *SessionManager) SetTitle(key, title string) error
  func (m *SessionManager) SetAlias(fromKey, toKey string) error
  func (m *SessionManager) ResolveAlias(key string) string
  ```

- [ ] Session key format:
  ```go
  // Telegram: telegram:123456789
  // Web: web:uuid-v4
  func MakeSessionKey(channel bus.ChannelType, id string) string {
      return fmt.Sprintf("%s:%s", channel, id)
  }

  func ParseSessionKey(key string) (channel bus.ChannelType, id string) {
      parts := strings.SplitN(key, ":", 2)
      if len(parts) != 2 {
          return "", key
      }
      return bus.ChannelType(parts[0]), parts[1]
  }
  ```

---

### 3.2 Bidirectional Sync

**Файлы:**
```
internal/gateway/
  telegram_sync.go
```

**Задачи:**

- [ ] Telegram → UI sync:
  ```go
  // В Telegram adapter при получении сообщения:
  func (c *TelegramConnector) handleUpdate(update telego.Update) {
      msg := bus.NewInboundMessage(
          bus.ChannelTypeTelegram,
          userID,
          MakeSessionKey(bus.ChannelTypeTelegram, chatID),
          text,
          metadata,
      )

      // Publish to bus (agent loop обработает)
      c.bus.PublishInbound(*msg)

      // ALSO broadcast to WebSocket для real-time UI
      c.bus.PublishSystem(bus.SystemEvent{
          Type:      "telegram_message",
          SessionID: msg.SessionID,
          Data:      msg.ToMap(),
      })
  }
  ```

- [ ] UI → Telegram sync:
  ```go
  // Agent loop генерирует OutboundMessage
  // Broadcast отправляет в Telegram И WebSocket одновременно
  func (a *AgentLoop) sendResponse(msg bus.OutboundMessage) {
      // Broadcast to all channels
      a.bus.BroadcastOutbound(msg, bus.ChannelTypeAPI) // exclude API
  }
  ```

- [ ] Message correlation:
  ```go
  type MessageContext struct {
      OriginalSessionID string
      ReplyToMessageID  string
      Source            string
  }
  ```

---

### 3.3 Session Aliases

**Задачи:**

- [ ] Позволяет связать Telegram чат с Web сессией:
  ```go
  // Пользователь в Telegram пишет "/connect web"
  // Создаётся alias: telegram:123456 → web:abc-123

  func (m *SessionManager) CreateWebSessionForTelegram(telegramKey string) (*Session, error) {
      webKey := MakeSessionKey(bus.ChannelTypeWeb, uuid.New().String())
      session := &Session{
          ID:           webKey,
          CanonicalKey: webKey,
          Source:       "web",
      }
      m.SetAlias(telegramKey, webKey)
      return session, nil
  }
  ```

---

## Phase 4: Commands & Polish

**Длительность:** 1 неделя
**Приоритет:** P2 (средний)
**Зависимости:** Phase 2, Phase 3

### 4.1 Cross-Channel Commands

**Файлы:**
```
internal/commands/
  handler.go
  handler_test.go
  commands/
    new.go
    sessions.go
    resume.go
    help.go
    clear.go
```

**Задачи:**

- [ ] Command handler:
  ```go
  type CommandHandler struct {
      sessionManager *gateway.SessionManager
      bus            *bus.MessageBus
  }

  func (h *CommandHandler) IsCommand(content string) bool
  func (h *CommandHandler) Handle(msg bus.InboundMessage) (*bus.OutboundMessage, bool)
  ```

- [ ] Commands:
  ```go
  // /new - start new session
  // /sessions - list sessions
  // /resume <id> - resume session
  // /help - show help
  // /clear - clear current session
  // /rename <title> - rename session
  // /status - show status
  ```

- [ ] Integration в agent loop:
  ```go
  func (l *AgentLoop) ProcessMessage(msg bus.InboundMessage) {
      // 1. Check for commands FIRST
      if l.cmdHandler.IsCommand(msg.Content) {
          resp, handled := l.cmdHandler.Handle(msg)
          if handled && resp != nil {
              l.bus.PublishOutbound(*resp)
              return // Don't invoke LLM
          }
      }

      // 2. Process with LLM
      // ...
  }
  ```

---

### 4.2 LLM Streaming Integration

**Файлы:**
```
internal/llm/
  streaming.go
  streaming_test.go
```

**Задачи:**

- [ ] Streaming callback:
  ```go
  type StreamCallback func(chunk string, done bool)

  func (p *ZaiProvider) StreamChat(ctx context.Context, messages []Message, callback StreamCallback) error
  ```

- [ ] Integration в agent loop:
  ```go
  func (l *AgentLoop) processWithLLM(msg bus.InboundMessage) {
      // Emit stream_start
      l.bus.PublishSystem(bus.NewSystemEvent(
          bus.SystemEventStreamStart,
          msg.SessionID,
          nil,
      ))

      // Stream response
      err := p.StreamChat(ctx, messages, func(chunk string, done bool) {
          if done {
              // Emit stream_end
              l.bus.PublishSystem(bus.NewSystemEvent(
                  bus.SystemEventStreamEnd,
                  msg.SessionID,
                  nil,
              ))
          } else {
              // Emit stream_chunk
              l.bus.PublishSystem(bus.NewSystemEvent(
                  bus.SystemEventStreamChunk,
                  msg.SessionID,
                  map[string]any{"content": chunk},
              ))
          }
      })
  }
  ```

---

### 4.3 Error Handling & UX

**Задачи:**

- [ ] Connection lost handling:
  ```javascript
  // In websocket.js
  on(type, callback) {
      switch(type) {
          case 'close':
              Alpine.store('app').connected = false
              this.showReconnectUI()
              break
          case 'open':
              Alpine.store('app').connected = true
              this.hideReconnectUI()
              break
      }
  }

  showReconnectUI() {
      // Show banner: "Connection lost. Reconnecting..."
  }
  ```

- [ ] Error notifications:
  ```javascript
  function showError(message, details) {
      Alpine.store('app').notifications.push({
          type: 'error',
          message,
          details,
          timestamp: Date.now()
      })
  }
  ```

- [ ] Loading states:
  ```html
  <button :disabled="sending" :class="{ 'opacity-50': sending }">
      <i data-lucide="loader-2" class="animate-spin" x-show="sending"></i>
      <span x-text="sending ? 'Sending...' : 'Send'"></span>
  </button>
  ```

---

## Phase 5: Optional Enhancements (v2)

**Длительность:** TBD
**Приоритет:** P3 (низкий)
**Зависимости:** Phase 4

### 5.1 Authentication

**Задачи:**

- [ ] Token-based auth для WebSocket
- [ ] Session cookies для HTTP
- [ ] First-run setup wizard
- [ ] API key support

### 5.2 Tunnel Support

**Задачи:**

- [ ] Cloudflare tunnel integration
- [ ] Auto-install cloudflared
- [ ] Remote access из интернета
- [ ] Public URL display в UI

### 5.3 Mission Control (Multi-Agent)

**Задачи:**

- [ ] Agent profiles (roles, capabilities)
- [ ] Task management
- [ ] Activity feed
- [ ] Real-time coordination

### 5.4 PWA Support

**Задачи:**

- [ ] manifest.json
- [ ] Service worker
- [ ] Offline support
- [ ] Install prompt

---

## Timeline Summary

| Phase | Длительность | Зависимости | Результат                              |
| ----- | ------------ | ----------- | -------------------------------------- |
| 0     | 1 неделя     | -           | SystemEvent bus, ChannelAdapter        |
| 1     | 1-2 недели   | Phase 0     | HTTP/WebSocket server, REST API        |
| 2     | 1-2 недели   | Phase 1     | Web UI с chat и sessions               |
| 3     | 1 неделя     | Phase 1, 2  | Telegram ↔ UI sync                     |
| 4     | 1 неделя     | Phase 2, 3  | Commands, streaming, polish            |
| 5     | TBD          | Phase 4     | Auth, tunnel, mission control, PWA     |
| **Total** | **5-7 недель**   |             |                                        |

---

## Milestones

### M1: Basic Gateway (Phase 0 + 1.1-1.3)
**Цель:** Работающий HTTP сервер с WebSocket
**Критерий:** `curl localhost:8080/api/health` возвращает `{"status":"ok"}`

### M2: UI Prototype (Phase 1.4 + 2.1)
**Цель:** Базовый UI с WebSocket подключением
**Критерий:** Открытие localhost:8080 показывает chat interface

### M3: Full Chat (Phase 2.2-2.4 + 3.1)
**Цель:** Полноценный чат в UI
**Критерий:** Можно отправить сообщение и получить ответ от агента

### M4: Telegram Sync (Phase 3.2-3.3)
**Цель:** Telegram ↔ UI bidirectional sync
**Критерий:** Сообщение в Telegram появляется в UI и наоборот

### M5: Production Ready (Phase 4)
**Цель:** Готовый к использованию dashboard
**Критерий:** Все команды работают, streaming работает, errors обрабатываются

---

## Configuration Example

```toml
# config.toml

[gateway]
enabled = true
host = "127.0.0.1"
port = 8080
tls_enabled = false
origins = ["http://localhost:8080", "http://127.0.0.1:8080"]

[telegram]
enabled = true
token = "${TELEGRAM_BOT_TOKEN}"
whitelist = [123456789]

[llm]
provider = "zai"
model = "glm-4-flash"
api_key = "${ZAI_API_KEY}"

[agent]
max_iterations = 10
timeout_seconds = 300
```

---

## Next Steps

1. **Review & Approve** — обсудить план, внести корректировки
2. **Phase 0 Start** — начать с SystemEvent bus extension
3. **Weekly Sync** — еженедельный прогресс review
4. **Iterate** — адаптировать план по мере реализации

---

*Plan generated: 2026-02-16*
*Based on: PocketPaw, Moltis architecture analysis*
