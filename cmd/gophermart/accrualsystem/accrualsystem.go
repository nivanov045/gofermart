package accrualsystem

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/nivanov045/gofermart/internal/order"
)

type accrualsystem struct {
	databasePath     string
	isDebug          bool
	channelToService chan<- order.Order
	ordersToProcess  []string
	mutexToOrders    sync.Mutex
}

func New(databasePath string, isDebug bool) (*accrualsystem, error) {
	res := &accrualsystem{
		databasePath:    databasePath,
		isDebug:         isDebug,
		ordersToProcess: []string{},
	}
	go res.processOrders()
	return res, nil
}

func (a *accrualsystem) processOrders() {
	for {
		a.mutexToOrders.Lock()
		if len(a.ordersToProcess) > 0 {
			log.Println("accrual::processOrders::info: added")
			go a.getAccrual(a.ordersToProcess[0])
			a.ordersToProcess = a.ordersToProcess[1:]
		}
		a.mutexToOrders.Unlock()
		time.Sleep(100 * time.Millisecond)
	}
}

func (a *accrualsystem) SetChannelToResponseToService(ch chan order.Order) {
	a.channelToService = ch
}

func (a *accrualsystem) addToOrders(orderNumber string) {
	a.mutexToOrders.Lock()
	a.ordersToProcess = append(a.ordersToProcess, orderNumber)
	a.mutexToOrders.Unlock()
}

func (a *accrualsystem) RunListenToService(channelFromService <-chan string) {
	log.Println("accrual::RunListenToService::info: started")
	ctx := context.Background()
	for {
		select {
		case <-ctx.Done():
			return
		case ord := <-channelFromService:
			log.Println("accrual::RunListenToService::info: received value")
			a.addToOrders(ord)
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

func (a *accrualsystem) getAccrual(orderNumber string) {
	if a.isDebug {
		var res order.Order
		res.Number = orderNumber
		random := rand.Intn(10)
		if random < 2 {
			log.Println("accrual::getAccrual::info: NEW:", orderNumber)
			time.Sleep(1 * time.Second)
			a.addToOrders(orderNumber)
			return
		}
		if random < 3 {
			log.Println("accrual::getAccrual::info: PROCESSING:", orderNumber)
			res.Status = order.ProcessingTypeProcessing
		} else if random < 4 {
			log.Println("accrual::getAccrual::info: INVALID:", orderNumber)
			res.Status = order.ProcessingTypeInvalid
		} else {
			log.Println("accrual::getAccrual::info: PROCESSED:", orderNumber)
			res.Status = order.ProcessingTypeProcessed
			res.Accrual = int64(random * 1000)
		}
		a.channelToService <- res
		return
	}

	client := &http.Client{}
	requestURL := a.databasePath + "/api/orders/" + orderNumber
	request, err := http.NewRequest(http.MethodGet, requestURL, bytes.NewBuffer([]byte(orderNumber)))
	if err != nil {
		log.Println("accrual::getAccrual::error: NewRequest:", err)
		a.mutexToOrders.Lock()
		a.ordersToProcess = append(a.ordersToProcess, orderNumber)
		a.mutexToOrders.Unlock()
		return
	}
	request.Header.Set("Content-Type", "text/html")
	response, err := client.Do(request)
	if err != nil {
		log.Println("accrual::getAccrual::error: Do:", err)
		a.addToOrders(orderNumber)
		return
	}
	switch response.StatusCode {
	case http.StatusOK:
		defer response.Body.Close()
		respBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println("accrual::getAccrual::error: ReadAll:", err)
			a.addToOrders(orderNumber)
			return
		}
		var res order.InterfaceForAccrualSystem
		err = json.Unmarshal(respBody, &res)
		if err != nil {
			log.Println("accrual::getAccrual::error: Unmarshal:", err)
			a.addToOrders(orderNumber)
			return
		}
		resAsOrder := order.Order{
			Number: res.Number,
			Status: res.Status,
		}
		if res.Status == order.ProcessingTypeProcessed {
			resAsOrder.Accrual = int64(res.Accrual * 100)
		}
		a.channelToService <- resAsOrder
	case http.StatusTooManyRequests:
		retryAfter := response.Header.Get("Retry-After")
		n, err := strconv.ParseInt(retryAfter, 10, 64)
		if err != nil {
			log.Println("accrual::getAccrual::error: ParseInt:", err)
			return
		}
		time.Sleep(time.Duration(n) * time.Second)
		a.addToOrders(orderNumber)
	default:
		log.Println("accrual::getAccrual::info: default")
		defer response.Body.Close()
		respBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println("accrual::getAccrual::error: ReadAll:", err)
			a.addToOrders(orderNumber)
			return
		}
		log.Println("accrual::getAccrual::info:", string(respBody))
		a.addToOrders(orderNumber)
	}
}
