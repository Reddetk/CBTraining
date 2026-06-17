//go:build testing
// +build testing

package testhelpers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Reddetk/CBTraining/core/entity"
)

// ============================================================================
// СИНХРОНИЗАЦИЯ И ОЖИДАНИЕ
// ============================================================================

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

// ============================================================================
// АНАЛИЗ ПОРЯДКА ОБРАБОТКИ
// ============================================================================

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
func (m *MockPaymentProcessor) VisualizeTime(log *MockLogger) {
	processed := m.GetProcessed()
	if len(processed) == 0 {
		log.Info("VisualizeTime: no processed payments")
		return
	}

	// Находим временные границы
	minT := processed[0].StartTime
	maxT := processed[0].EndTime
	for _, r := range processed {
		if r.StartTime.Before(minT) {
			minT = r.StartTime
		}
		if r.EndTime.After(maxT) {
			maxT = r.EndTime
		}
	}

	const width = 60 // ширина временной шкалы в символах
	totalDur := maxT.Sub(minT)

	// Строим поле: каждая строка — один платёж
	type row struct {
		label string
		line  string
	}

	rows := make([]row, len(processed))
	maxLabel := 0

	for i, r := range processed {
		label := fmt.Sprintf("#%d ETE:%-12s", i+1, r.ETE)
		if len(label) > maxLabel {
			maxLabel = len(label)
		}

		startOff := int(float64(r.StartTime.Sub(minT)) / float64(totalDur) * width)
		endOff := int(float64(r.EndTime.Sub(minT)) / float64(totalDur) * width)
		if endOff <= startOff {
			endOff = startOff + 1
		}

		bar := make([]byte, width)
		for j := range bar {
			bar[j] = ' '
		}
		// Зона ожидания (от 0 до start)
		for j := 0; j < startOff && j < width; j++ {
			bar[j] = '·'
		}
		// Зона выполнения
		char := byte('=')
		if r.Error != nil {
			char = '+'
		}
		for j := startOff; j < endOff && j < width; j++ {
			bar[j] = char
		}

		suffix := fmt.Sprintf(" +%dms", r.Duration.Milliseconds())
		if r.Error != nil {
			suffix += " ERR"
		}

		rows[i] = row{
			label: label,
			line:  string(bar) + suffix,
		}
	}

	// Шапка с временно́й шкалой
	header := fmt.Sprintf("%*s  |", maxLabel, "")
	ticks := "0ms"
	mid := fmt.Sprintf("%dms", totalDur.Milliseconds()/2)
	end := fmt.Sprintf("%dms", totalDur.Milliseconds())
	scale := fmt.Sprintf("%-*s%s%*s", width/3, ticks, mid, width-width/3-len(mid)-len(end)+len(end), end)

	separator := fmt.Sprintf("%s--+-%s", strings.Repeat("-", maxLabel), strings.Repeat("-", width+12))

	log.Info("=== Payment Timeline ===")
	log.Info(header + scale)
	log.Info(separator)

	for _, r := range rows {
		log.Info(fmt.Sprintf("%-*s  | %s", maxLabel, r.label, r.line))
	}

	log.Info(separator)

	// Итоговая статистика
	log.Info(fmt.Sprintf("Total time: %dms | Payments: %d",
		totalDur.Milliseconds(),
		len(processed),
	))
}

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
