//go:build testing
// +build testing

package testhelpers

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Reddetk/CBTraining/logger"
)

// ============================================================================
// ГРАФ КОНФЛИКТОВ
// ============================================================================

type ConflictEdge struct {
	ETEA        string
	ETEB        string
	SharedIBANs []string
}

type ConflictGraph struct {
	records []*ProcessRecord
	edges   []ConflictEdge
	adj     map[int][]int // adj[i] = индексы записей конфликтующих с records[i]
}

func BuildConflictGraph(records []*ProcessRecord) *ConflictGraph {
	g := &ConflictGraph{
		records: records,
		adj:     make(map[int][]int),
	}
	for i := 0; i < len(records); i++ {
		for j := i + 1; j < len(records); j++ {
			shared := sharedIBANs(records[i], records[j])
			if len(shared) > 0 {
				g.edges = append(g.edges, ConflictEdge{
					ETEA:        records[i].ETE,
					ETEB:        records[j].ETE,
					SharedIBANs: shared,
				})
				g.adj[i] = append(g.adj[i], j)
				g.adj[j] = append(g.adj[j], i)
			}
		}
	}
	return g
}

func sharedIBANs(a, b *ProcessRecord) []string {
	setA := map[string]bool{a.DebtorIBAN: true, a.CreditorIBAN: true}
	var shared []string
	for _, iban := range []string{b.DebtorIBAN, b.CreditorIBAN} {
		if setA[iban] {
			shared = append(shared, iban)
		}
	}
	return shared
}

func intervalsOverlap(a, b *ProcessRecord) bool {
	return a.StartTime.Before(b.EndTime) && b.StartTime.Before(a.EndTime)
}

func isConflicting(g *ConflictGraph, i, j int) bool {
	for _, nb := range g.adj[i] {
		if nb == j {
			return true
		}
	}
	return false
}

// AssertErrorNotNil проверяет что ошибка не нулевая
func AssertErrorNotNil(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected error, got nil", msg)
	}
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

// ============================================================================
// ИНВАРИАНТ 1: SAFETY
// ============================================================================

type SafetyViolation struct {
	RecA        *ProcessRecord
	RecB        *ProcessRecord
	SharedIBANs []string
	OverlapMs   int64
}

func (v SafetyViolation) String() string {
	return fmt.Sprintf(
		"SAFETY VIOLATION: ETE:%-12s [%s – %s]\n"+
			"                  ETE:%-12s [%s – %s]\n"+
			"                  shared IBANs: %v | overlap: %dms",
		v.RecA.ETE, v.RecA.StartTime.Format("15:04:05.000"), v.RecA.EndTime.Format("15:04:05.000"),
		v.RecB.ETE, v.RecB.StartTime.Format("15:04:05.000"), v.RecB.EndTime.Format("15:04:05.000"),
		v.SharedIBANs, v.OverlapMs,
	)
}

func CheckSafety(g *ConflictGraph) []SafetyViolation {
	idx := make(map[string]int, len(g.records))
	for i, r := range g.records {
		idx[r.ETE] = i
	}
	var violations []SafetyViolation
	for _, edge := range g.edges {
		a := g.records[idx[edge.ETEA]]
		b := g.records[idx[edge.ETEB]]
		if !intervalsOverlap(a, b) {
			continue
		}
		overlapStart := a.StartTime
		if b.StartTime.After(overlapStart) {
			overlapStart = b.StartTime
		}
		overlapEnd := a.EndTime
		if b.EndTime.Before(overlapEnd) {
			overlapEnd = b.EndTime
		}
		violations = append(violations, SafetyViolation{
			RecA:        a,
			RecB:        b,
			SharedIBANs: edge.SharedIBANs,
			OverlapMs:   overlapEnd.Sub(overlapStart).Milliseconds(),
		})
	}
	return violations
}

func AssertSafety(t *testing.T, g *ConflictGraph) bool {
	t.Helper()
	violations := CheckSafety(g)
	if len(violations) == 0 {
		t.Log("+ Safety: no conflicting accounts ran in parallel")
		return true
	}
	for _, v := range violations {
		t.Errorf("- %s", v)
	}
	return false
}

// ============================================================================
// ИНВАРИАНТ 2: LIVENESS
// ============================================================================

func CheckLiveness(records []*ProcessRecord, submittedETEs []string) []string {
	processed := make(map[string]bool, len(records))
	for _, r := range records {
		processed[r.ETE] = true
	}
	var missing []string
	for _, ete := range submittedETEs {
		if !processed[ete] {
			missing = append(missing, ete)
		}
	}
	return missing
}

func AssertLiveness(t *testing.T, records []*ProcessRecord, submittedETEs []string) bool {
	t.Helper()
	missing := CheckLiveness(records, submittedETEs)
	if len(missing) == 0 {
		t.Logf("+ Liveness: all %d payments processed", len(submittedETEs))
		return true
	}
	for _, ete := range missing {
		t.Errorf("- Liveness starvation: ETE:%s submitted but never processed", ete)
	}
	return false
}

func WaitLiveness(t *testing.T, m *MockPaymentProcessor, submittedETEs []string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().UTC().Add(timeout)
	for time.Now().UTC().Before(deadline) {
		if len(CheckLiveness(m.GetProcessed(), submittedETEs)) == 0 {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	missing := CheckLiveness(m.GetProcessed(), submittedETEs)
	t.Logf("WaitLiveness timeout — missing: %v", missing)
	return false
}

// ============================================================================
// ИНВАРИАНТ 3: PARALLELISM
// ============================================================================

type ParallelPair struct {
	RecA      *ProcessRecord
	RecB      *ProcessRecord
	OverlapMs int64
}

func CheckParallelism(g *ConflictGraph) (pairs []ParallelPair, hasIndependentPairs bool) {
	for i := 0; i < len(g.records); i++ {
		for j := i + 1; j < len(g.records); j++ {
			if isConflicting(g, i, j) {
				continue
			}
			hasIndependentPairs = true
			a, b := g.records[i], g.records[j]
			if !intervalsOverlap(a, b) {
				continue
			}
			overlapStart := a.StartTime
			if b.StartTime.After(overlapStart) {
				overlapStart = b.StartTime
			}
			overlapEnd := a.EndTime
			if b.EndTime.Before(overlapEnd) {
				overlapEnd = b.EndTime
			}
			pairs = append(pairs, ParallelPair{
				RecA:      a,
				RecB:      b,
				OverlapMs: overlapEnd.Sub(overlapStart).Milliseconds(),
			})
		}
	}
	return
}

func AssertParallelism(t *testing.T, g *ConflictGraph) bool {
	t.Helper()
	pairs, hasIndependent := CheckParallelism(g)
	switch {
	case !hasIndependent:
		t.Log("~ Parallelism: skipped — all payments share accounts")
		return true
	case len(pairs) > 0:
		t.Logf("+ Parallelism: %d independent pair(s) ran in parallel", len(pairs))
		for _, p := range pairs {
			t.Logf("  ETE:%-12s ∥ ETE:%-12s overlap: %dms", p.RecA.ETE, p.RecB.ETE, p.OverlapMs)
		}
		return true
	default:
		t.Error("- Parallelism: independent payments exist but all were serialized")
		return false
	}
}

// ============================================================================
// КОМБО: ВСЕ ИНВАРИАНТЫ РАЗОМ
// ============================================================================

func AssertAllInvariants(t *testing.T, m *MockPaymentProcessor, submittedETEs []string) {
	t.Helper()
	records := m.GetProcessed()
	g := BuildConflictGraph(records)

	t.Log("─────────────────────────────────────────")
	t.Log("          INVARIANT CHECK")
	t.Log("─────────────────────────────────────────")
	AssertSafety(t, g)
	AssertLiveness(t, records, submittedETEs)
	AssertParallelism(t, g)
	t.Log("─────────────────────────────────────────")
}

// ============================================================================
// ВИЗУАЛИЗАЦИЯ + ИНВАРИАНТЫ В ЛОГАХ
// ============================================================================

func (m *MockPaymentProcessor) VisualizeTime(log logger.Logger) {
	processed := m.GetProcessed()
	if len(processed) == 0 {
		log.Info("VisualizeTime: no processed payments")
		return
	}

	minT, maxT := processed[0].StartTime, processed[0].EndTime
	for _, r := range processed {
		if r.StartTime.Before(minT) {
			minT = r.StartTime
		}
		if r.EndTime.After(maxT) {
			maxT = r.EndTime
		}
	}

	const width = 60
	totalDur := maxT.Sub(minT)
	if totalDur == 0 {
		totalDur = time.Millisecond
	}

	g := BuildConflictGraph(processed)

	// Отмечаем нарушителей Safety
	violators := make(map[string]bool)
	for _, v := range CheckSafety(g) {
		violators[v.RecA.ETE] = true
		violators[v.RecB.ETE] = true
	}

	type row struct{ label, line string }
	rows := make([]row, len(processed))
	maxLabel := 0

	for i, r := range processed {
		label := fmt.Sprintf("#%d %-8s %s → %s",
			i+1, truncate(r.ETE[:2], 10), shortIBAN(r.DebtorIBAN), shortIBAN(r.CreditorIBAN))
		if len(label) > maxLabel {
			maxLabel = len(label)
		}

		startOff := clamp(int(float64(r.StartTime.Sub(minT))/float64(totalDur)*width), 0, width-1)
		endOff := clamp(int(float64(r.EndTime.Sub(minT))/float64(totalDur)*width), startOff+1, width)

		bar := make([]byte, width)
		for j := range bar {
			bar[j] = ' '
		}
		for j := 0; j < startOff; j++ {
			bar[j] = '.'
		}
		ch := byte('=')
		if r.Error != nil {
			ch = 'X'
		} else if violators[r.ETE] {
			ch = '!'
		}
		for j := startOff; j < endOff; j++ {
			bar[j] = ch
		}

		suffix := fmt.Sprintf(" %4dms", r.Duration.Milliseconds())
		if r.Error != nil {
			suffix += " [ERR]"
		}
		if violators[r.ETE] {
			suffix += " [!SAFETY]"
		}
		rows[i] = row{label, string(bar) + suffix}
	}

	totalMs := totalDur.Milliseconds()
	sep := strings.Repeat("─", maxLabel+2) + "─┼─" + strings.Repeat("─", width+10)
	timeScale := fmt.Sprintf("0ms%s%dms%s%dms",
		strings.Repeat(" ", width/3-3), totalMs/2,
		strings.Repeat(" ", width/3-len(fmt.Sprint(totalMs/2))), totalMs)

	log.Info("═══════════════ Payment Timeline ═══════════════")
	log.Info(fmt.Sprintf("Legend: [=] ok  [X] error  [!] safety violation  [.] waiting"))
	log.Info(fmt.Sprintf("%*s  │ %s", maxLabel, "time→", timeScale))
	log.Info(sep)
	for _, r := range rows {
		log.Info(fmt.Sprintf("%-*s  │ %s", maxLabel, r.label, r.line))
	}
	log.Info(sep)

	// Граф конфликтов
	if len(g.edges) > 0 {
		log.Info("Conflict edges:")
		for _, e := range g.edges {
			log.Info(fmt.Sprintf("  ETE:%-12s ── ETE:%-12s  IBANs: %v", e.ETEA, e.ETEB, e.SharedIBANs))
		}
	}

	// Инварианты
	log.Info("─────────── Invariants ───────────")
	safetyViolations := CheckSafety(g)
	if len(safetyViolations) == 0 {
		log.Info("+ Safety:      OK")
	} else {
		for _, v := range safetyViolations {
			log.Info(fmt.Sprintf("- Safety:      %s", v))
		}
	}

	pairs, hasIndependent := CheckParallelism(g)
	switch {
	case !hasIndependent:
		log.Info("~ Parallelism: no independent pairs")
	case len(pairs) > 0:
		log.Info(fmt.Sprintf("+ Parallelism: %d pair(s) ran in parallel", len(pairs)))
	default:
		log.Info("- Parallelism: independent payments were serialized")
	}

	errors := 0
	for _, r := range processed {
		if r.Error != nil {
			errors++
		}
	}
	log.Info(fmt.Sprintf("Wall time: %dms | payments: %d | errors: %d",
		totalMs, len(processed), errors))
}

func shortIBAN(iban string) string {
	if len(iban) <= 8 {
		return iban
	}
	return iban[:4] + ".." + iban[len(iban)-4:]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// DebugProcessedRecords выводит список обработанных записей для диагностики таймаутов
func DebugProcessedRecords(m *MockPaymentProcessor) string {
	records := m.GetProcessed()
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("processed %d records:\n", len(records)))
	for _, r := range records {
		sb.WriteString(fmt.Sprintf("  ETE:%-12s start:%s dur:%dms err:%v\n",
			r.ETE, r.StartTime.Format("15:04:05.000"), r.Duration.Milliseconds(), r.Error))
	}
	return sb.String()
}

// VerifyProcessingOrder проверяет что в цепочке конфликтующих платежей
// каждый следующий стартовал после завершения предыдущего
func VerifyProcessingOrder(t *testing.T, m *MockPaymentProcessor, etes []string) bool {
	t.Helper()
	records := m.GetProcessed()
	idx := make(map[string]*ProcessRecord, len(records))
	for _, r := range records {
		idx[r.ETE] = r
	}

	g := BuildConflictGraph(records)
	ok := true

	// Проверяем только рёбра графа конфликтов
	for _, edge := range g.edges {
		a := idx[edge.ETEA]
		b := idx[edge.ETEB]
		if a == nil || b == nil {
			continue
		}
		// Один должен завершиться до старта другого
		aBeforeB := !a.EndTime.After(b.StartTime)
		bBeforeA := !b.EndTime.After(a.StartTime)
		if !aBeforeB && !bBeforeA {
			t.Errorf("ORDER VIOLATION on conflict edge: ETE:%-14s [end:%s] and ETE:%-14s [start:%s] overlap — shared IBANs: %v",
				edge.ETEA, a.EndTime.Format("15:04:05.000"),
				edge.ETEB, b.StartTime.Format("15:04:05.000"),
				edge.SharedIBANs,
			)
			ok = false
		}
	}
	return ok
}
