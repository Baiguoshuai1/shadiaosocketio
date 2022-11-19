package shadiaosocketio

import (
	"errors"
	"sync"
)

var (
	ErrorWaiterNotFound = errors.New("Waiter not found")
)

/**
Processes functions that require answers, also known as acknowledge or ack
*/
type ackProcessor struct {
	counter          int
	counterLock      sync.Mutex
	resultWaitersMap sync.Map
}

/**
get next id of ack call
*/
func (a *ackProcessor) getNextId() int {
	a.counterLock.Lock()
	defer a.counterLock.Unlock()

	a.counter++
	return a.counter
}

/**
Just before the ack function called, the waiter should be added
to wait and receive response to ack call
*/
func (a *ackProcessor) addWaiter(id int, w chan interface{}) {
	a.resultWaitersMap.Store(id, w)
}

/**
removes waiter that is unnecessary anymore
*/
func (a *ackProcessor) removeWaiter(id int) {
	a.resultWaitersMap.Delete(id)
}

/**
check if waiter with given ack id is exists, and returns it
*/
func (a *ackProcessor) getWaiter(id int) (chan interface{}, error) {
	if waiter, ok := a.resultWaitersMap.Load(id); ok {
		return waiter.(chan interface{}), nil
	}
	return nil, ErrorWaiterNotFound
}
