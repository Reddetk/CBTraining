//go:build testing
// +build testing

// Package testhelpers содержит helper'ы и mock'и для unit тестирования.
// Все файлы автоматически исключаются из production build'а благодаря build tags.
package testhelpers

import (
	"context"
	"fmt"
	"sync"
	"time"

	valobj "github.com/Reddetk/CBTraining/core/valObj"
	"github.com/Reddetk/CBTraining/logger"
	outport "github.com/Reddetk/CBTraining/ports/outports"
)

// ============================================================================
// MOCK PAYMENT PROCESSOR
// ============================================================================

// ProcessRecord хранит информацию о обработанном платеже для анализа
type ProcessRecord struct {
	ETE          string        // ETE транзакции
	StartTime    time.Time     // Время начала обработки
	EndTime      time.Time     // Время окончания обработки
	Duration     time.Duration // Длительность обработки
	Error        error         // Ошибка обработки, если была
	DebtorIBAN   string
	CreditorIBAN string
	Status       string
}

// MockPaymentProcessor имитирует обработку платежей с контролируемой задержкой
type MockPaymentProcessor struct {
	mu        sync.Mutex
	processed []*ProcessRecord
	delay     time.Duration // имитируемая длительность обработки
	orderSeq  int64         // счётчик для отслеживания порядка
}

// NewMockPaymentProcessor создаёт новый mock ProcessPayment с заданной задержкой
func NewMockPaymentProcessor(delay time.Duration) *MockPaymentProcessor {
	return &MockPaymentProcessor{
		processed: make([]*ProcessRecord, 0),
		delay:     delay,
	}
}

// ProcessTX имитирует обработку платежа
func (m *MockPaymentProcessor) ProcessTX(ctx context.Context, req outport.TXRequest) error {
	record := &ProcessRecord{
		ETE:          req.EndToEndIdentification,
		StartTime:    time.Now().UTC(),
		DebtorIBAN:   req.DebtorIBAN,
		CreditorIBAN: req.CreditorIBAN,
		Status:       string(valobj.Processing),
	}

	// Имитируем обработку с задержкой
	select {
	case <-time.After(m.delay):
	case <-ctx.Done():
		return ctx.Err()
	}

	record.EndTime = time.Now().UTC()
	record.Duration = record.EndTime.Sub(record.StartTime)
	record.Status = string(valobj.Completed)

	m.mu.Lock()
	m.processed = append(m.processed, record)
	m.mu.Unlock()

	return record.Error
}

func (m *MockPaymentProcessor) GetProcessed() []*ProcessRecord {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.processed
}

func (m *MockPaymentProcessor) GetProcessedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.processed)
}

// ============================================================================
// MOCK PENDING REPO
// ============================================================================

// MockPendingRepo имитирует хранилище отложенных платежей
type MockPendingRepo struct {
	mu       sync.Mutex
	pending  []outport.TXRecord
	loadErr  error
	saveErr  error
	isLoaded bool
}

// NewMockPendingRepo создаёт новый mock репо для отложенных платежей
func NewMockPendingRepo() *MockPendingRepo {
	return &MockPendingRepo{
		pending: make([]outport.TXRecord, 0),
	}
}

// const pandRecTXsCount = 1

// // LoadPendingPayments загружает отложенные платежи
// func (m *MockPendingRepo) LoadPendingPayments(ctx context.Context) ([]outport.TXRecord, error) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	_, blds := BuildDependentChain(pandRecTXsCount)
// 	pTxRecs := make([]outport.TXRecord, 0, pandRecTXsCount)

// 	for _, bld := range blds {
// 		pTxRec := bld.BuiderToTXrec()
// 		pTxRecs = append(pTxRecs, pTxRec)
// 	}

// 	m.isLoaded = true
// 	return pTxRecs, nil
// }

func (m *MockPendingRepo) LoadPendingPayments(ctx context.Context) ([]outport.TXRecord, error) {
	return []outport.TXRecord{}, nil
}

// SetLoadError устанавливает ошибку для LoadPendingPayments
func (m *MockPendingRepo) SetLoadError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadErr = err
}

// ============================================================================
// MOCK PAYMENT REPO
// ============================================================================

// MockPaymentRepo имитирует хранилище платежей
type MockPaymentRepo struct {
	mu        sync.Mutex
	payments  map[string]outport.TXRecord
	insertErr error
	queryErr  error
}

// NewMockPaymentRepo создаёт новый mock репо для платежей
func NewMockPaymentRepo() *MockPaymentRepo {
	return &MockPaymentRepo{
		payments: make(map[string]outport.TXRecord),
	}
}

// InsertTX сохраняет новый платёж
func (m *MockPaymentRepo) InsertTX(ctx context.Context, tx outport.TXRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.insertErr != nil {
		return m.insertErr
	}
	m.payments[tx.TXID] = tx
	return nil
}

// GetTXByID загружает платёж по ID
func (m *MockPaymentRepo) GetTXByID(ctx context.Context, txID string) (*outport.TXRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	if tx, exists := m.payments[txID]; exists {
		return &tx, fmt.Errorf("such tx dose not exist, error of get")
	}
	return nil, nil
}

func (m *MockPaymentRepo) PersistProcessResult(ctx context.Context, TXID, result string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.insertErr != nil {
		return m.insertErr
	}
	tx, exists := m.payments[TXID]
	if !exists {
		return fmt.Errorf("such tx dose not exist, eroor of persist")
	}
	tx.Status = result
	m.payments[TXID] = tx
	return nil
}

// GetAllPayments возвращает все сохранённые платежи
func (m *MockPaymentRepo) GetAllPayments() []outport.TXRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]outport.TXRecord, 0, len(m.payments))
	for _, tx := range m.payments {
		result = append(result, tx)
	}
	return result
}

// ============================================================================
// MOCK LOGGER
// ============================================================================

// LogEntry представляет одну запись логирования
type LogEntry struct {
	Level     string
	Message   string
	Timestamp time.Time
	Fields    []logger.Field
}

// MockLogger имитирует логирование и реализует интерфейс logger.Logger
type MockLogger struct {
	mu      sync.Mutex
	entries []*LogEntry
}

// NewMockLogger создаёт новый mock logger
func NewMockLogger() *MockLogger {
	return &MockLogger{
		entries: make([]*LogEntry, 0),
	}
}

// Info логирует информационное сообщение
func (m *MockLogger) Info(msg string, fields ...logger.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := &LogEntry{
		Level:     "info",
		Message:   msg,
		Timestamp: time.Now().UTC(),
		// копируем слайс, чтобы защититься от мутаций снаружи
		Fields: append([]logger.Field(nil), fields...),
	}
	m.entries = append(m.entries, entry)
}

// Error логирует сообщение об ошибке
func (m *MockLogger) Error(msg string, fields ...logger.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := &LogEntry{
		Level:     "error",
		Message:   msg,
		Timestamp: time.Now().UTC(),
		Fields:    append([]logger.Field(nil), fields...),
	}
	m.entries = append(m.entries, entry)
}

// Дополнительные уровни логирования — только для удобства тестов.
// Боевой интерфейс их не требует, поэтому они на сигнатуру logger.Logger не влияют.

func (m *MockLogger) Warn(msg string, fields ...logger.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := &LogEntry{
		Level:     "warn",
		Message:   msg,
		Timestamp: time.Now().UTC(),
		Fields:    append([]logger.Field(nil), fields...),
	}
	m.entries = append(m.entries, entry)
}

func (m *MockLogger) Debug(msg string, fields ...logger.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := &LogEntry{
		Level:     "debug",
		Message:   msg,
		Timestamp: time.Now().UTC(),
		Fields:    append([]logger.Field(nil), fields...),
	}
	m.entries = append(m.entries, entry)
}

// GetEntries возвращает все логирования
func (m *MockLogger) GetEntries() []*LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]*LogEntry, len(m.entries))
	copy(result, m.entries)
	return result
}

// GetErrorCount возвращает количество ошибок
func (m *MockLogger) GetErrorCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, entry := range m.entries {
		if entry.Level == "error" {
			count++
		}
	}
	return count
}
