package outport

import "context"

type OutboxWorker interface {
	Run(ctx context.Context)
}
