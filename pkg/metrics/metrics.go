package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type WebSocketMetrics struct {
	ActiveConnections        prometheus.Gauge
	ConnectionsTotal         prometheus.Counter
	ConnectionDuration       prometheus.Histogram
	UnexpectedCloseCount     prometheus.Counter
	AuthenticationErrorCount prometheus.Counter

	MessagesSent          *prometheus.CounterVec
	MessagesReceived      *prometheus.CounterVec
	BytesSent             prometheus.Counter
	BytesReceived         prometheus.Counter
	MessageLatency        prometheus.Histogram
	MessageBufferSize     prometheus.Gauge
	MessageBufferOverflow prometheus.Counter
}

type KafkaMetrics struct {
	MessagesProcessed *prometheus.CounterVec
	ProcessingRate    *prometheus.GaugeVec
	ConsumerLag       *prometheus.GaugeVec
	DeserializeErrors prometheus.Counter
	KafkaErrors       *prometheus.CounterVec
}

type RedisMetrics struct {
	ClientRegistrations   prometheus.Counter
	ClientUnregistrations prometheus.Counter
	MessagePublications   prometheus.Counter
	RedisOperationLatency prometheus.Histogram
	RedisOperationErrors  *prometheus.CounterVec
}

type HttpMetrics struct {
	RequestsTotal      *prometheus.CounterVec
	ResponseStatusCode *prometheus.CounterVec
	RequestDuration    *prometheus.HistogramVec
}

type SystemMetrics struct {
	CPUUsage       prometheus.Gauge
	MemoryUsage    prometheus.Gauge
	GoroutineCount prometheus.Gauge
	GCDuration     prometheus.Histogram
	GCCount        prometheus.Counter
}

type BusinessMetrics struct {
	MessagesByType *prometheus.CounterVec
	MessagesByUser *prometheus.CounterVec
}

type Metrics struct {
	WebSocket WebSocketMetrics
	Kafka     KafkaMetrics
	Redis     RedisMetrics
	Http      HttpMetrics
	System    SystemMetrics
	Business  BusinessMetrics
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		WebSocket: WebSocketMetrics{
			ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "websocket_active_connections",
				Help:      "Количество активных WebSocket соединений",
			}),
			ConnectionsTotal: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "websocket_connections_total",
				Help:      "Общее количество установленных WebSocket соединений",
			}),
			ConnectionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "websocket_connection_duration_seconds",
				Help:      "Длительность WebSocket соединений в секундах",
				Buckets:   prometheus.ExponentialBuckets(1, 2, 10),
			}),
			UnexpectedCloseCount: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "websocket_unexpected_close_total",
				Help:      "Количество неожиданно закрытых соединений",
			}),
			AuthenticationErrorCount: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "websocket_auth_errors_total",
				Help:      "Количество ошибок аутентификации",
			}),
			MessagesSent: promauto.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "websocket_messages_sent_total",
				Help:      "Количество отправленных сообщений, по типам",
			}, []string{"message_type"}),
			MessagesReceived: promauto.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "websocket_messages_received_total",
				Help:      "Количество полученных сообщений, по типам",
			}, []string{"message_type"}),
			BytesSent: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "websocket_bytes_sent_total",
				Help:      "Количество отправленных байт",
			}),
			BytesReceived: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "websocket_bytes_received_total",
				Help:      "Количество полученных байт",
			}),
			MessageLatency: promauto.NewHistogram(prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "websocket_message_latency_seconds",
				Help:      "Время от получения сообщения из Kafka до отправки клиенту",
				Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
			}),
			MessageBufferSize: promauto.NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "websocket_message_buffer_size",
				Help:      "Текущий размер буфера сообщений",
			}),
			MessageBufferOverflow: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "websocket_message_buffer_overflow_total",
				Help:      "Количество переполнений буфера сообщений",
			}),
		},
		Kafka: KafkaMetrics{
			MessagesProcessed: promauto.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "kafka_messages_processed_total",
				Help:      "Количество обработанных сообщений из Kafka, по темам",
			}, []string{"topic"}),
			ProcessingRate: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "kafka_processing_rate",
				Help:      "Скорость обработки сообщений из Kafka (сообщений/сек), по темам",
			}, []string{"topic"}),
			ConsumerLag: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "kafka_consumer_lag",
				Help:      "Отставание консюмера Kafka (разница между последним доступным и обработанным сообщением), по темам и партициям",
			}, []string{"topic", "partition"}),
			DeserializeErrors: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "kafka_deserialize_errors_total",
				Help:      "Количество ошибок десериализации сообщений из Kafka",
			}),
			KafkaErrors: promauto.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "kafka_errors_total",
				Help:      "Количество ошибок Kafka, по кодам ошибок",
			}, []string{"code"}),
		},
		Redis: RedisMetrics{
			ClientRegistrations: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "redis_client_registrations_total",
				Help:      "Количество регистраций клиентов в Redis",
			}),
			ClientUnregistrations: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "redis_client_unregistrations_total",
				Help:      "Количество удалений клиентов из Redis",
			}),
			MessagePublications: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "redis_message_publications_total",
				Help:      "Количество публикаций сообщений в Redis",
			}),
			RedisOperationLatency: promauto.NewHistogram(prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "redis_operation_latency_seconds",
				Help:      "Время выполнения операций с Redis",
				Buckets:   prometheus.ExponentialBuckets(0.001, 2, 8),
			}),
			RedisOperationErrors: promauto.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "redis_operation_errors_total",
				Help:      "Количество ошибок операций с Redis, по типам",
			}, []string{"operation"}),
		},
		Http: HttpMetrics{
			RequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Количество HTTP-запросов, по методам и путям",
			}, []string{"method", "path"}),
			ResponseStatusCode: promauto.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_response_status_code_total",
				Help:      "Количество HTTP-ответов, по кодам статуса",
			}, []string{"status_code"}),
			RequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "Время обработки HTTP-запроса",
				Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
			}, []string{"path"}),
		},
		System: SystemMetrics{
			CPUUsage: promauto.NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "system_cpu_usage",
				Help:      "Использование CPU приложением (в процентах)",
			}),
			MemoryUsage: promauto.NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "system_memory_usage_bytes",
				Help:      "Использование памяти приложением (в байтах)",
			}),
			GoroutineCount: promauto.NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "system_goroutine_count",
				Help:      "Количество активных горутин",
			}),
			GCDuration: promauto.NewHistogram(prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "system_gc_duration_seconds",
				Help:      "Длительность сборки мусора",
				Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 10),
			}),
			GCCount: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "system_gc_count_total",
				Help:      "Количество запусков сборки мусора",
			}),
		},
		Business: BusinessMetrics{
			MessagesByType: promauto.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "business_messages_by_type_total",
				Help:      "Количество сообщений по типам",
			}, []string{"message_type"}),
			MessagesByUser: promauto.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "business_messages_by_user_total",
				Help:      "Количество сообщений по пользователям",
			}, []string{"user_id"}),
		},
	}

	return m
}
