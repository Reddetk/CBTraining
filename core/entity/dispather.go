package entity

import (
	"context"
	"sync"
)

// Dispatcher — маршрутизатор TX по связным компонентам IBAN-графа.
// Гарантирует: TX с пересекающимися IBAN попадают в одну группу
// и обрабатываются строго последовательно одним воркером.
// TX из разных групп обрабатываются параллельно.
type Dispatcher struct {
	mu sync.Mutex // защищает все поля ниже, кроме process и ctx

	// ibanToGroup — индекс для O(1) поиска группы по IBAN.
	// Все IBAN одной связной компоненты графа указывают на один *Group.
	// Инвариант: после merge gB→gA все ключи бывшего gB переключены на gA.
	ibanToGroup map[string]*Group

	// semaphore — ограничивает число живых воркеров (= число активных групп).
	// Слот захватывается при создании группы, освобождается когда воркер завершается.
	semaphore chan struct{}

	// process — единственная точка где происходит реальная работа:
	// PublishEvent → DeletePending → UpsertAudit.
	// Dispatcher не знает деталей — только вызывает и ждёт завершения.
	process func(tx *PaymentTX) error

	// ctx — сигнал graceful shutdown для всего диспетчера.
	// Отмена ctx останавливает приём новых TX и даёт воркерам дочитать буферы.
	ctx context.Context

	// wg — барьер на Shutdown(): блокирует пока все воркеры не вышли из runWorker.
	wg sync.WaitGroup
}

func NewDispatcher(ctx context.Context, maxWorkers int, process func(tx *PaymentTX) error) *Dispatcher {
	return &Dispatcher{
		ibanToGroup: make(map[string]*Group),
		semaphore:   make(chan struct{}, maxWorkers),
		process:     process,
		ctx:         ctx,
	}
}
