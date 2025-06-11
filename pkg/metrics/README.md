# Metrics

Описание всех метрик, собираемых сервисом WebSocket Gateway.

## WebSocket Metrics

| Метрика | Тип | Описание |
|---------|-----|----------|
| `websocket_active_connections` | Gauge | Количество активных WebSocket соединений |
| `websocket_connections_total` | Counter | Общее количество установленных WebSocket соединений |
| `websocket_connection_duration_seconds` | Histogram | Длительность WebSocket соединений в секундах |
| `websocket_unexpected_close_total` | Counter | Количество неожиданно закрытых соединений |
| `websocket_auth_errors_total` | Counter | Количество ошибок аутентификации |
| `websocket_messages_sent_total` | Counter Vec | Количество отправленных сообщений (по типам) |
| `websocket_messages_received_total` | Counter Vec | Количество полученных сообщений (по типам) |
| `websocket_bytes_sent_total` | Counter | Количество отправленных байт |
| `websocket_bytes_received_total` | Counter | Количество полученных байт |
| `websocket_message_latency_seconds` | Histogram | Время от получения сообщения из Kafka до отправки клиенту |
| `websocket_message_buffer_size` | Gauge | Текущий размер буфера сообщений |
| `websocket_message_buffer_overflow_total` | Counter | Количество переполнений буфера сообщений |

## Kafka Metrics

| Метрика | Тип | Описание |
|---------|-----|----------|
| `kafka_messages_processed_total` | Counter Vec | Количество обработанных сообщений из Kafka (по темам) |
| `kafka_processing_rate` | Gauge Vec | Скорость обработки сообщений (сообщений/сек, по темам) |
| `kafka_consumer_lag` | Gauge Vec | Отставание консюмера (по темам и партициям) |
| `kafka_deserialize_errors_total` | Counter | Количество ошибок десериализации сообщений |
| `kafka_errors_total` | Counter Vec | Количество ошибок Kafka (по кодам ошибок) |

## Redis Metrics

| Метрика | Тип | Описание |
|---------|-----|----------|
| `redis_client_registrations_total` | Counter | Количество регистраций клиентов в Redis |
| `redis_client_unregistrations_total` | Counter | Количество удалений клиентов из Redis |
| `redis_message_publications_total` | Counter | Количество публикаций сообщений в Redis |
| `redis_operation_latency_seconds` | Histogram | Время выполнения операций с Redis |
| `redis_operation_errors_total` | Counter Vec | Количество ошибок операций с Redis (по типам) |

## HTTP Metrics

| Метрика | Тип | Описание |
|---------|-----|----------|
| `http_requests_total` | Counter Vec | Количество HTTP-запросов (по методам и путям) |
| `http_response_status_code_total` | Counter Vec | Количество HTTP-ответов (по кодам статуса) |
| `http_request_duration_seconds` | Histogram Vec | Время обработки HTTP-запроса (по путям) |

## System Metrics

| Метрика | Тип | Описание |
|---------|-----|----------|
| `system_cpu_usage` | Gauge | Использование CPU приложением (в процентах) |
| `system_memory_usage_bytes` | Gauge | Использование памяти приложением (в байтах) |
| `system_goroutine_count` | Gauge | Количество активных горутин |
| `system_gc_duration_seconds` | Histogram | Длительность сборки мусора |
| `system_gc_count_total` | Counter | Количество запусков сборки мусора |

## Business Metrics

| Метрика | Тип | Описание |
|---------|-----|----------|
| `business_messages_by_type_total` | Counter Vec | Количество сообщений по типам |
| `business_messages_by_user_total` | Counter Vec | Количество сообщений по пользователям | 