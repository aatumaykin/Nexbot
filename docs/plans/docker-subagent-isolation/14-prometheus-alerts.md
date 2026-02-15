# Этап 14: Prometheus Alerting (MVP)

## Цель

4 критичных alerting rules для мониторинга Docker пула.

## Файлы

### `prometheus/alerts.yml`

```yaml
groups:
  - name: subagent
    rules:
      # 1. Container down (CRITICAL)
      - alert: SubagentContainerDown
        expr: nexbot_subagent_containers_total == 0
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "No subagent containers running"
          description: "All containers are down. Subagent functionality unavailable."
      
      # 2. High error rate (WARNING)
      - alert: SubagentHighErrorRate
        expr: |
          rate(nexbot_subagent_tasks_total{status="error"}[5m])
          /
          rate(nexbot_subagent_tasks_total[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High subagent error rate"
          description: "{{ $value | humanizePercentage }} of tasks are failing."
      
      # 3. Pool exhausted (WARNING)
      - alert: SubagentPoolExhausted
        expr: nexbot_subagent_containers_active >= nexbot_subagent_containers_total
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Subagent pool exhausted"
          description: "All containers are busy. New requests will queue."
      
      # 4. Circuit breaker open (INFO)
      - alert: SubagentCircuitBreakerOpen
        expr: nexbot_subagent_circuit_breaker_state == 1
        for: 1m
        labels:
          severity: info
        annotations:
          summary: "Circuit breaker open"
          description: "Pool circuit breaker is open. Requests are being rejected."
```

## Grafana Dashboard (базовый)

```json
{
  "dashboard": {
    "title": "Nexbot Subagent",
    "panels": [
      {
        "title": "Containers",
        "type": "stat",
        "targets": [
          {"expr": "nexbot_subagent_containers_total", "legendFormat": "Total"},
          {"expr": "nexbot_subagent_containers_active", "legendFormat": "Active"}
        ]
      },
      {
        "title": "Tasks/s",
        "type": "timeseries",
        "targets": [
          {"expr": "rate(nexbot_subagent_tasks_total{status=\"success\"}[5m])", "legendFormat": "Success"},
          {"expr": "rate(nexbot_subagent_tasks_total{status=\"error\"}[5m])", "legendFormat": "Error"}
        ]
      },
      {
        "title": "Latency (P95)",
        "type": "stat",
        "targets": [
          {"expr": "histogram_quantile(0.95, rate(nexbot_subagent_task_duration_seconds_bucket[5m]))"}
        ]
      },
      {
        "title": "Circuit Breaker",
        "type": "gauge",
        "targets": [
          {"expr": "nexbot_subagent_circuit_breaker_state"}
        ],
        "fieldConfig": {
          "defaults": {
            "mappings": [
              {"value": 0, "text": "Closed"},
              {"value": 1, "text": "Open"},
              {"value": 2, "text": "Half-Open"}
            ]
          }
        }
      }
    ]
  }
}
```

## 6 базовых метрик

| Метрик                          | Тип      | Описание                    |
| ------------------------------- | -------- | --------------------------- |
| `nexbot_subagent_containers_total` | Gauge    | Всего контейнеров           |
| `nexbot_subagent_containers_active` | Gauge    | Активных контейнеров        |
| `nexbot_subagent_tasks_total`   | Counter  | Задачи по статусу           |
| `nexbot_subagent_task_duration_seconds` | Histogram | Длительность задач      |
| `nexbot_subagent_circuit_breaker_state` | Gauge    | Состояние CB (0/1/2)        |
| `nexbot_subagent_pending_requests` | Gauge    | Задач в очереди             |

## Ключевые решения (MVP)

1. **4 alerts** — только критичные (container_down, high_error, pool_exhausted, cb_open)
2. **6 metrics** — базовый coverage
3. **Severity-based** — critical/warning/info
4. **For duration** — избежать flapping
