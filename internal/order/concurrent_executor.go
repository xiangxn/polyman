package order

import (
	"context"
	"log"
	"sync"

	"github.com/xiangxn/polyman/internal/config"
	"github.com/xiangxn/polyman/internal/model"

	"github.com/xiangxn/go-polymarket-sdk/orders"
	pm "github.com/xiangxn/go-polymarket-sdk/polymarket"
)

type ConcurrentExecutor struct {
	ch      chan model.Intent
	workers int
	wg      sync.WaitGroup
	cache   sync.Map

	pmClient *pm.PolymarketClient
}

type OrderKey struct {
	Token string
	Side  orders.SideType
}

func NewConcurrentExecutor(pmClient *pm.PolymarketClient, config config.OrderEngineConfig) *ConcurrentExecutor {
	return &ConcurrentExecutor{
		ch:       make(chan model.Intent, config.QueueSize),
		workers:  config.WorkerNum,
		pmClient: pmClient,
	}
}

// Submit 投递订单
func (e *ConcurrentExecutor) Submit(ctx context.Context, intent model.Intent) error {
	select {
	case e.ch <- intent:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Run 启动 worker goroutine
func (e *ConcurrentExecutor) Run(ctx context.Context) error {
	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go func(id int) {
			defer e.wg.Done()
			log.Printf("[Executor] worker-%d 开始", id)
			for {
				select {
				case <-ctx.Done():
					log.Printf("[Executor] worker-%d 停止", id)
					return
				case in := <-e.ch:
					key := OrderKey{Token: in.Token, Side: in.Side}
					// 原子去重
					if _, exists := e.cache.LoadOrStore(key, struct{}{}); exists {
						log.Printf("[Executor] worker-%d 重复订单跳过: %v", id, key)
						continue
					}
					e.handleOrder(id, in, key)
				}
			}
		}(i)
	}

	// 等待所有 worker 停止
	e.wg.Wait()
	return ctx.Err()
}

// handleOrder 处理订单逻辑（打印 / 调用下单）
func (e *ConcurrentExecutor) handleOrder(workerID int, intent model.Intent, key OrderKey) {
	// ⚠️ 如果这里有共享状态，比如 position，需要加锁
	log.Printf("[Executor] worker-%d processing order: %+v", workerID, intent)

	// // 下单逻辑
	// if e.pmClient != nil {
	// 	// 创建订单
	// 	side := pmModel.BUY
	// 	if intent.Side == orders.SELL {
	// 		side = pmModel.SELL
	// 	}
	// 	order, err := e.pmClient.CreateOrder(&orders.UserOrder{
	// 		TokenID: intent.Token,
	// 		Price:   intent.Price,
	// 		Size:    intent.Size,
	// 		Side:    side,
	// 	}, orders.CreateOrderOptions{
	// 		TickSize:      orders.TickSize001,
	// 		SignatureType: pmModel.POLY_GNOSIS_SAFE,
	// 	})
	// 	if err != nil {
	// 		log.Printf("[Worker %d] 下单失败: %v", workerID, err)
	// 		return
	// 	}
	// 	// 发送订单
	// 	result, err := e.pmClient.PostOrder(order, intent.OrderType, false)
	// 	if err != nil {
	// 		log.Printf("[Worker %d] 下单失败: %v", workerID, err)
	// 		return
	// 	}
	// 	orderId := result.Get("orderId").String()
	// 	log.Printf("[Worker %d] 下单成功: %v", workerID, orderId)
	// } else {
	// 	log.Printf("[Worker %d] 下单失败: %v", workerID, "pmClient is nil")
	// }

	// 下单完成后释放缓存
	e.cache.Delete(key)
}
