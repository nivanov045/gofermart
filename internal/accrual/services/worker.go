package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/nivanov045/gofermart/internal/accrual/log"
	"github.com/nivanov045/gofermart/internal/accrual/models"
)

func fanOut(inputCh chan models.OrderList, n int) []chan models.OrderList {
	chs := make([]chan models.OrderList, 0, n)
	for i := 0; i < n; i++ {
		ch := make(chan models.OrderList)
		chs = append(chs, ch)
	}

	go func() {
		defer func(chs []chan models.OrderList) {
			for _, ch := range chs {
				close(ch)
			}
		}(chs)

		for i := 0; ; i = (i + 1) % n {
			order, ok := <-inputCh
			if !ok {
				return
			}

			ch := chs[i]
			ch <- order
		}
	}()

	return chs
}

func fanIn(inputChs ...chan accrualResult) chan accrualResult {
	outCh := make(chan accrualResult)

	go func() {
		wg := &sync.WaitGroup{}

		for _, inputCh := range inputChs {
			wg.Add(1)

			go func(inputCh chan accrualResult) {
				defer wg.Done()
				for item := range inputCh {
					outCh <- item
				}
			}(inputCh)
		}

		wg.Wait()
		close(outCh)
	}()

	return outCh
}

func (s *Service) newWorker(ctx context.Context, in chan models.OrderList, out chan accrualResult) {
	go func() {
		for order := range in {
			log.Debug(fmt.Sprintf("Order '%v' start processing", order.ID))
			err := s.storage.UpdateOrderStatus(ctx, order.ID, models.OrderStatus{Status: models.OrderStatusProcessing})
			if err != nil {
				log.Error(err)
			}

			accrual := s.computeAccrual(ctx, order)
			out <- accrual
		}
	}()
}

func (s *Service) computeAccrual(ctx context.Context, order models.OrderList) accrualResult {
	accrual := 0.0
	for _, orderProduct := range order.Goods {
		products, err := s.storage.MatchProducts(ctx, orderProduct.Description)
		if err != nil {
			return accrualResult{id: order.ID, err: err}
		}

		for _, product := range products {
			switch product.RewardType {
			case models.RewardTypePoints:
				accrual += product.Reward
			case models.RewardTypePercent:
				accrual += 0.01 * product.Reward * orderProduct.Price
			default:
				return accrualResult{id: order.ID, err: fmt.Errorf("unknown reward type: '%v'", product.RewardType)}
			}
		}
	}

	return accrualResult{id: order.ID, accrual: accrual, err: nil}
}

func (s *Service) process(ctx context.Context, workerChs []chan accrualResult) {
	for resultAccrual := range fanIn(workerChs...) {
		if resultAccrual.err != nil {
			log.Error(resultAccrual.err)
			continue
		}

		err := s.storage.UpdateOrderStatus(ctx, resultAccrual.id, models.OrderStatus{Status: models.OrderStatusProcessed, Accrual: resultAccrual.accrual})
		if err != nil {
			log.Error(err)

			err := s.storage.UpdateOrderStatus(ctx, resultAccrual.id, models.OrderStatus{Status: models.OrderStatusRegistered, Accrual: 0})
			if err != nil {
				log.Error(err)
			}
			continue
		}
		log.Debug(fmt.Sprintf("Order '%v' processed", resultAccrual.id))

		err = s.queue.RemoveOrder(ctx, resultAccrual.id)
		if err != nil {
			log.Error(err)
		}
	}
}
