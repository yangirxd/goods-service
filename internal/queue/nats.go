package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/yangirxd/goods-service/internal/clickhouse"
)

type Logger struct {
	nc *nats.Conn
}

type LogConsumer struct {
	nc     *nats.Conn
	ch     *clickhouse.Client
	stopCh chan struct{}
}

func NewLogger(url string) (*Logger, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connect to nats: %w", err)
	}

	return &Logger{nc: nc}, nil
}

func (l *Logger) Log(action string, entityID int64, data interface{}) error {
	event := clickhouse.LogEvent{
		Action:    action,
		Timestamp: time.Now(),
		EntityID:  entityID,
		Data:      data,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	err = l.nc.Publish("goods.logs", payload)
	if err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	return nil
}

func (l *Logger) Close() {
	l.nc.Close()
}

// NewLogConsumer создает новый экземпляр потребителя логов
func NewLogConsumer(natsURL, clickhouseURL string) (*LogConsumer, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("подключение к NATS: %w", err)
	}

	ch, err := clickhouse.NewClient(clickhouseURL)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("подключение к ClickHouse: %w", err)
	}

	return &LogConsumer{
		nc:     nc,
		ch:     ch,
		stopCh: make(chan struct{}),
	}, nil
}

// Start запускает обработку сообщений из NATS и запись в ClickHouse
func (c *LogConsumer) Start() error {
	c.ch.Start()

	sub, err := c.nc.Subscribe("goods.logs", func(msg *nats.Msg) {
		var event clickhouse.LogEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			fmt.Printf("Ошибка разбора сообщения: %v\n", err)
			return
		}

		if err := c.ch.AddEvent(&event); err != nil {
			fmt.Printf("Ошибка добавления события: %v\n", err)
		}
	})
	if err != nil {
		return fmt.Errorf("ошибка подписки на goods.logs: %w", err)
	}

	<-c.stopCh
	sub.Unsubscribe()
	return nil
}

// Stop останавливает обработку и записывает оставшиеся логи
func (c *LogConsumer) Stop() error {
	close(c.stopCh)
	return c.ch.Stop()
}

// Close закрывает все соединения
func (c *LogConsumer) Close() error {
	if err := c.Stop(); err != nil {
		return err
	}
	c.nc.Close()
	return c.ch.Close()
}
