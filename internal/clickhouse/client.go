package clickhouse

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

const batchSize = 100
const flushInterval = 5 * time.Second

type LogEvent struct {
	Action    string      `json:"action"`
	Timestamp time.Time   `json:"timestamp"`
	EntityID  int64       `json:"entity_id"`
	Data      interface{} `json:"data"`
}

type Client struct {
	db     *sql.DB
	batch  []*LogEvent
	mu     sync.Mutex
	stopCh chan struct{}
}

// NewClient создает новый экземпляр клиента ClickHouse
func NewClient(clickhouseURL string) (*Client, error) {
	db, err := sql.Open("clickhouse", clickhouseURL)
	if err != nil {
		return nil, fmt.Errorf("подключение к ClickHouse: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("проверка подключения к ClickHouse: %w", err)
	}

	return &Client{
		db:     db,
		batch:  make([]*LogEvent, 0, batchSize),
		stopCh: make(chan struct{}),
	}, nil
}

// Start запускает обработку батчей и периодическую запись в ClickHouse
func (c *Client) Start() {
	go c.flushLoop()
}

// Stop останавливает обработку и записывает оставшиеся логи
func (c *Client) Stop() error {
	close(c.stopCh)
	return c.flush()
}

// Close закрывает соединение с ClickHouse
func (c *Client) Close() error {
	if err := c.Stop(); err != nil {
		return err
	}
	return c.db.Close()
}

// AddEvent добавляет событие в пакет и записывает его, если пакет заполнен
func (c *Client) AddEvent(event *LogEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.batch = append(c.batch, event)
	if len(c.batch) >= batchSize {
		return c.flush()
	}
	return nil
}

// flushLoop периодически записывает накопленные логи
func (c *Client) flushLoop() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			if len(c.batch) > 0 {
				if err := c.flush(); err != nil {
					fmt.Printf("Ошибка периодической записи: %v\n", err)
				}
			}
			c.mu.Unlock()
		case <-c.stopCh:
			return
		}
	}
}

// flush записывает накопленные логи в ClickHouse
func (c *Client) flush() error {
	if len(c.batch) == 0 {
		return nil
	}

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("начало транзакции: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO logs.goods_events (action, timestamp, entity_id, data)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("подготовка запроса: %w", err)
	}
	defer stmt.Close()

	for _, event := range c.batch {
		data, err := json.Marshal(event.Data)
		if err != nil {
			fmt.Printf("Ошибка сериализации данных: %v\n", err)
			continue
		}

		_, err = stmt.Exec(
			event.Action,
			event.Timestamp,
			event.EntityID,
			string(data),
		)
		if err != nil {
			return fmt.Errorf("выполнение запроса: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("подтверждение транзакции: %w", err)
	}

	// Очищаем пакет после успешной записи
	c.batch = c.batch[:0]
	return nil
}
