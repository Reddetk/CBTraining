package entity

import (
	"context"
	"sync/atomic"

	"github.com/Reddetk/CBTraining/core/consts"
)

// Group — связный компонент графа IBAN.
// Объединяет все TX у которых есть хотя бы один общий ЦС с любым другим TX группы.
//
// Инварианты которые Dispatcher обязан соблюдать:
//   1. Ровно один живой воркер читает g.ch — никакой параллельной обработки внутри группы.
//   2. TX поступают в g.ch в порядке вызовов Add() — FIFO внутри группы.
//   3. Новые TX не отправляются в g.ch после вызова workerCancelSig() —
//      только дрейн уже лежащих в буфере.
//
// Жизненный цикл группы:
//   создана (newGroup) → воркер запущен → [TX обрабатываются]
//   → merge в другую группу (workerCancelSig вызван) → воркер дочитывает буфер → завершается
//   ИЛИ
//   → dispatcher.Shutdown() (parent ctx отменён) → воркер дочитывает буфер → завершается
type Group struct {
	Ch chan *PaymentTX

	// ctx — контекст жизни этой конкретной группы.
	// Отменяется в двух случаях:
	//   а) Dispatcher.Add() решил слить эту группу в другую (merge) — вызывает workerCancelSig
	//   б) parent ctx диспетчера отменён (graceful shutdown) — отменяет все дочерние ctx автоматически
	// Воркер слушает ctx.Done() чтобы узнать что пора завершаться.
	GrCtx context.Context

	// workerCancelSig — функция отмены ctx этой группы.
	// Вызывается только Dispatcher-ом при merge: сигнализирует воркеру
	// что новые TX больше не придут и можно дочитать буфер и выйти.
	// После вызова workerCancelSig Dispatcher перестаёт писать в g.ch.
	WorkerCancelSig context.CancelFunc

	Merging    atomic.Bool
	DrainReady chan struct{}
}

// NewGroup создаёт новую группу привязанную к жизненному циклу parent-контекста.
// parent — это ctx диспетчера: когда диспетчер завершается,
// все дочерние группы получают сигнал остановки автоматически.
func NewGroup(parent context.Context) *Group {
	// WithCancel даёт два рычага управления жизнью воркера:
	//   ctx.Done()          — воркер слушает этот канал
	//   workerCancelSig()   — Dispatcher дёргает при merge или shutdown
	ctx, workerCancelSig := context.WithCancel(parent)
	return &Group{
		Ch:              make(chan *PaymentTX, consts.ChanBuffer),
		GrCtx:           ctx,
		WorkerCancelSig: workerCancelSig,
		DrainReady:      make(chan struct{}),
	}
}
