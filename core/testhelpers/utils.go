//go:build testing
// +build testing

package testhelpers

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Reddetk/CBTraining/core/entity"
)

// ============================================================================
// СИНХРОНИЗАЦИЯ И ОЖИДАНИЕ
// ============================================================================

// WaitForGroupCount ждёт пока в dispatcher'е появится N групп
func WaitForGroupCount(t *testing.T, d *entity.Dispatcher, expected int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for {
		d.Mu.Lock()
		current := len(d.IbanToGroup)
		d.Mu.Unlock()

		if current == expected {
			return true
		}

		if time.Now().After(deadline) {
			t.Logf("timeout waiting for %d groups, got %d", expected, current)
			return false
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// WaitForProcessedCount ждёт пока mock processor обработает N платежей
func WaitForProcessedCount(t *testing.T, m *MockPaymentProcessor, expected int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for {
		if m.GetProcessedCount() >= expected {
			return true
		}

		if time.Now().After(deadline) {
			t.Logf("timeout waiting for %d processed, got %d", expected, m.GetProcessedCount())
			return false
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// ============================================================================
// ИЗМЕРЕНИЕ ВРЕМЕНИ
// ============================================================================

// TimedExecution выполняет функцию и возвращает её длительность
func TimedExecution(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// AssertTimeWithinRange проверяет что actual время находится в диапазоне
func AssertTimeWithinRange(t *testing.T, actual, expectedMin, expectedMax time.Duration, msg string) {
	if actual < expectedMin || actual > expectedMax {
		t.Errorf("%s: expected duration between %v and %v, got %v",
			msg, expectedMin, expectedMax, actual)
	}
}

// AssertTimeDiffWithinPercent проверяет что time1 и time2 отличаются не более чем на percent процентов
func AssertTimeDiffWithinPercent(t *testing.T, time1, time2 time.Duration, percent float64, msg string) {
	if time1 == 0 || time2 == 0 {
		t.Errorf("%s: cannot compare zero durations", msg)
		return
	}

	ratio := float64(time1) / float64(time2)
	if ratio < 1 {
		ratio = 1 / ratio
	}

	allowedRatio := 1.0 + (percent / 100.0)
	if ratio > allowedRatio {
		t.Errorf("%s: time difference exceeds %f%%. Expected ratio <= %f, got %f",
			msg, percent, allowedRatio, ratio)
	}
}

// MeasureLatency измеряет время выполнения функции
func MeasureLatency(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// ============================================================================
// АНАЛИЗ ПОРЯДКА ОБРАБОТКИ
// ============================================================================

// VerifyProcessingOrder проверяет что TX обработаны в ожидаемом порядке (FIFO)
func VerifyProcessingOrder(t *testing.T, m *MockPaymentProcessor, expectedOrder []string) bool {
	actual := m.GetProcessedOrder()

	if len(actual) != len(expectedOrder) {
		t.Errorf("processing order length mismatch: expected %d, got %d", len(expectedOrder), len(actual))
		return false
	}

	for i, expected := range expectedOrder {
		if actual[i] != expected {
			t.Errorf("processing order mismatch at position %d: expected %q, got %q", i, expected, actual[i])
			return false
		}
	}

	return true
}

// VerifyStartAfterEnd проверяет что tx2 начал обработку только ПОСЛЕ завершения tx1
func VerifyStartAfterEnd(t *testing.T, m *MockPaymentProcessor, ETE1, ETE2 string, maxGap time.Duration) bool {
	records := m.GetProcessed()

	var rec1, rec2 *ProcessRecord
	for _, rec := range records {
		if rec.ETE == ETE1 {
			rec1 = rec
		}
		if rec.ETE == ETE2 {
			rec2 = rec
		}
	}

	if rec1 == nil {
		t.Errorf("record not found for TX %q", ETE1)
		return false
	}
	if rec2 == nil {
		t.Errorf("record not found for TX %q", ETE2)
		return false
	}

	gap := rec2.StartTime.Sub(rec1.EndTime)

	if gap < 0 {
		t.Errorf("TX %q started before TX %q ended (gap: %v)", ETE2, ETE1, gap)
		return false
	}

	if gap > maxGap {
		t.Errorf("gap between TX %q end and TX %q start exceeds maximum (%v > %v)",
			ETE1, ETE2, gap, maxGap)
		return false
	}

	return true
}

// VerifyParallelExecution проверяет что две TX выполнялись параллельно
func VerifyParallelExecution(t *testing.T, m *MockPaymentProcessor, ETE1, ETE2 string) bool {
	records := m.GetProcessed()

	var rec1, rec2 *ProcessRecord
	for _, rec := range records {
		if rec.ETE == ETE1 {
			rec1 = rec
		}
		if rec.ETE == ETE2 {
			rec2 = rec
		}
	}

	if rec1 == nil || rec2 == nil {
		return false
	}

	// Проверяем что временные интервалы пересекаются
	if rec1.StartTime.Before(rec2.EndTime) && rec2.StartTime.Before(rec1.EndTime) {
		return true
	}

	t.Errorf("TX %q and %q did not execute in parallel", ETE1, ETE2)
	return false
}

// ============================================================================
// ПРОВЕРКА ГРУППИРОВКИ И LOCK'ОВ
// ============================================================================

// GetGroupCount возвращает текущее число групп в dispatcher'е
func GetGroupCount(d *entity.Dispatcher) int {
	d.Mu.Lock()
	defer d.Mu.Unlock()
	return len(d.IbanToGroup)
}

// VerifyIBANInSameGroup проверяет что все IBAN'ы из списка находятся в одной группе
func VerifyIBANInSameGroup(t *testing.T, d *entity.Dispatcher, ibans []string) bool {
	d.Mu.Lock()
	defer d.Mu.Unlock()

	if len(ibans) == 0 {
		return true
	}

	expectedGroup := d.IbanToGroup[ibans[0]]
	if expectedGroup == nil {
		t.Errorf("IBAN %q not found in dispatcher", ibans[0])
		return false
	}

	for _, iban := range ibans[1:] {
		group := d.IbanToGroup[iban]
		if group == nil {
			t.Errorf("IBAN %q not found in dispatcher", iban)
			return false
		}
		if group != expectedGroup {
			t.Errorf("IBAN %q in different group than %q", iban, ibans[0])
			return false
		}
	}

	return true
}

// VerifyIBANInDifferentGroups проверяет что IBAN'ы находятся в РАЗНЫХ группах
func VerifyIBANInDifferentGroups(t *testing.T, d *entity.Dispatcher, ibans []string) bool {
	d.Mu.Lock()
	defer d.Mu.Unlock()

	groups := make(map[*entity.Group]bool)

	for _, iban := range ibans {
		group := d.IbanToGroup[iban]
		if group == nil {
			t.Errorf("IBAN %q not found in dispatcher", iban)
			return false
		}
		if groups[group] {
			t.Errorf("IBAN %q in same group as another IBAN in the list", iban)
			return false
		}
		groups[group] = true
	}

	return true
}

// ============================================================================
// ВСПОМОГАТЕЛЬНЫЕ УТИЛИТЫ
// ============================================================================

// CreateContextWithTimeout создаёт context с timeout'ом
func CreateContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// AssertNilError проверяет что ошибка нулевая
func AssertNilError(t *testing.T, err error, msg string) {
	if err != nil {
		t.Errorf("%s: expected nil error, got %v", msg, err)
	}
}

// AssertErrorNotNil проверяет что ошибка не нулевая
func AssertErrorNotNil(t *testing.T, err error, msg string) {
	if err == nil {
		t.Errorf("%s: expected error, got nil", msg)
	}
}

// DebugDispatcherState выводит состояние dispatcher'а для отладки
func DebugDispatcherState(d *entity.Dispatcher) string {
	d.Mu.Lock()
	defer d.Mu.Unlock()

	uniqueGroups := make(map[*entity.Group]bool)
	for _, group := range d.IbanToGroup {
		uniqueGroups[group] = true
	}

	return fmt.Sprintf(
		"Dispatcher state: %d IBANs, %d unique groups",
		len(d.IbanToGroup),
		len(uniqueGroups),
	)
}

// DebugProcessedRecords выводит информацию о обработанных TX'ах
func DebugProcessedRecords(m *MockPaymentProcessor) string {
	records := m.GetProcessed()
	if len(records) == 0 {
		return "No processed records"
	}

	return fmt.Sprintf("Processed %d records: %v", len(records), m.GetProcessedOrder())
}

// WaitGroupAsync запускает функции асинхронно и ждёт их завершения
func WaitGroupAsync(fns ...func()) {
	var wg sync.WaitGroup
	for _, fn := range fns {
		wg.Add(1)
		go func(f func()) {
			defer wg.Done()
			f()
		}(fn)
	}
	wg.Wait()
}
