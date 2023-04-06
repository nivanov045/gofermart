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
	"time"

	"github.com/nivanov045/gofermart/internal/order"
)

type accrualsystem struct {
	databasePath     string
	isDebug          bool
	channelToService chan<- order.Order
	ordersToProcess  chan string
}

func New(databasePath string, isDebug bool) (*accrualsystem, error) {
	resultAccrualSystem := &accrualsystem{
		databasePath:    databasePath,
		isDebug:         isDebug,
		ordersToProcess: make(chan string),
	}
	go resultAccrualSystem.processOrders()
	return resultAccrualSystem, nil
}

func (a *accrualsystem) processOrders() {
	ctx := context.Background()
	for {
		select {
		case <-ctx.Done():
			return
		case ord := <-a.ordersToProcess:
			log.Println("accrual::processOrders::info: added")
			go a.getAccrual(ord)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (a *accrualsystem) SetChannelToResponseToService(ch chan order.Order) {
	a.channelToService = ch
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
			a.ordersToProcess <- ord
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

func (a *accrualsystem) getAccrual(orderNumber string) {
	if a.isDebug {
		var resultOrder order.Order
		resultOrder.Number = orderNumber
		random := rand.Intn(10)
		if random < 2 {
			log.Println("accrual::getAccrual::info: NEW:", orderNumber)
			time.Sleep(1 * time.Second)
			a.ordersToProcess <- orderNumber
			return
		}
		if random < 3 {
			log.Println("accrual::getAccrual::info: PROCESSING:", orderNumber)
			resultOrder.Status = order.ProcessingTypeProcessing
		} else if random < 4 {
			log.Println("accrual::getAccrual::info: INVALID:", orderNumber)
			resultOrder.Status = order.ProcessingTypeInvalid
		} else {
			log.Println("accrual::getAccrual::info: PROCESSED:", orderNumber)
			resultOrder.Status = order.ProcessingTypeProcessed
			resultOrder.Accrual = int64(random * 1000)
		}
		a.channelToService <- resultOrder
		return
	}

	client := &http.Client{}
	requestURL := a.databasePath + "/api/orders/" + orderNumber
	request, err := http.NewRequest(http.MethodGet, requestURL, bytes.NewBuffer([]byte(orderNumber)))
	if err != nil {
		log.Println("accrual::getAccrual::error: NewRequest:", err)
		a.ordersToProcess <- orderNumber
		return
	}
	request.Header.Set("Content-Type", "text/html")
	response, err := client.Do(request)
	if err != nil {
		log.Println("accrual::getAccrual::error: Do:", err)
		a.ordersToProcess <- orderNumber
		return
	}
	switch response.StatusCode {
	case http.StatusOK:
		defer response.Body.Close()
		respBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println("accrual::getAccrual::error: ReadAll:", err)
			a.ordersToProcess <- orderNumber
			return
		}
		var resultOrderInterface order.InterfaceForAccrualSystem
		err = json.Unmarshal(respBody, &resultOrderInterface)
		if err != nil {
			log.Println("accrual::getAccrual::error: Unmarshal:", err)
			a.ordersToProcess <- orderNumber
			return
		}
		resultAsOrder := order.Order{
			Number: resultOrderInterface.Number,
			Status: resultOrderInterface.Status,
		}
		if resultOrderInterface.Status == order.ProcessingTypeProcessed {
			resultAsOrder.Accrual = int64(resultOrderInterface.Accrual * 100)
		}
		a.channelToService <- resultAsOrder
	case http.StatusTooManyRequests:
		retryAfter := response.Header.Get("Retry-After")
		n, err := strconv.ParseInt(retryAfter, 10, 64)
		if err != nil {
			log.Println("accrual::getAccrual::error: ParseInt:", err)
			return
		}
		time.Sleep(time.Duration(n) * time.Second)
		a.ordersToProcess <- orderNumber
	default:
		log.Println("accrual::getAccrual::info: default")
		defer response.Body.Close()
		respBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println("accrual::getAccrual::error: ReadAll:", err)
			a.ordersToProcess <- orderNumber
			return
		}
		log.Println("accrual::getAccrual::info:", string(respBody))
		a.ordersToProcess <- orderNumber
	}
}
