package main

import (
	_ "github.com/Reddetk/CBTraining/adapters/primary/http"
	_ "github.com/gin-gonic/gin"
)

// @title           Payment Service API
// @version         1.0
// @description     Async payment processing service
// @host            localhost:8080
// @BasePath        /
func main() {
	// r := gin.Default()
	// h := http.NewHandler( /* inject deps */ )
	// http.RegisterRoutes(r, h)
	// r.Run(":8080")

	// rootCtx, cancel := context.WithCancel(context.Background())

	// svc, _ := core.NewPaymentManagerService(
	//     payProc, panRep, payRep,
	//     rootCtx, maxWorkers, log,
	// )

	// // первичный адаптер — HTTP сервер
	// srv := httpAdapter.New(svc)
	// go srv.ListenAndServe()

	// // ждём сигнала ОС
	// quit := make(chan os.Signal, 1)
	// signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// <-quit

	// log.Info("shutdown signal received")

	// // 1. Останавливаем HTTP — новые запросы не принимаем
	// httpCtx, httpCancel := context.WithTimeout(context.Background(), 30*time.Second)
	// defer httpCancel()
	// srv.Shutdown(httpCtx)

	// // 2. Отменяем rootCtx — воркеры получают GrCtx.Done()
	// cancel()

	// // 3. Ждём пока все воркеры дочитают буферы и выйдут
	// svc.Shutdown() // внутри — d.Wg.Wait()

	// log.Info("all workers stopped cleanly")
}
