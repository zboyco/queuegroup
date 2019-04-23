package queuegroup

import (
	"errors"
	"sync"
	"time"
)

// Groups 队列组
type Groups struct {
	timeout      int64 // 单个号业务办理超时时间（毫秒，默认不超时）
	expireSecond int64 // 组队列没有排号后多长时间关闭队列（秒，默认不关闭）
	queueMax     int64 // 单个队列最大长度（默认10）
	queues       map[int64]*Queue
	sync.Mutex
}

// Ticket 排队票据
type Ticket chan bool

// Queue 队列
type Queue struct {
	groupID       int64       // 组ID(窗口ID)
	ticketChan    chan Ticket // 排号队列
	currentTicket Ticket      // 当前办理号
	status        bool        // 队列状态(true:启用；false:关闭)
}

// 包级数据请求队列
var queueGroup *Groups

func init() {
	queueGroup = NewQueueGroup()
}

// NewQueueGroup 新建一个队列组
func NewQueueGroup() *Groups {
	return &Groups{
		queueMax: 10,
		queues:   make(map[int64]*Queue),
	}
}

// Config 设置参数
//
// queueMax 单个队列最大长度（默认10）
// timeout 单个号业务办理超时时间（毫秒，默认不超时）
// expireSecond 组队列没有排号后多长时间关闭队列（秒，默认不关闭）
func Config(queueMax, timeout, expireSecond int64) {
	queueGroup.Config(queueMax, timeout, expireSecond)
}

// GetQueue 获取组
//
// groupID 组号
func GetQueue(groupID int64) *Queue {
	return queueGroup.GetQueue(groupID)
}

// Config 设置参数
//
// queueMax 单个队列最大长度（默认10）
// timeout 单个号业务办理超时时间（毫秒，默认不超时）
// expireSecond 组队列没有排号后多长时间关闭队列（秒，默认不关闭）
func (g *Groups) Config(queueMax, timeout, expireSecond int64) {
	if queueMax > 0 {
		g.queueMax = queueMax
	}
	if timeout > 0 {
		g.timeout = timeout
	}
	if expireSecond > 0 {
		g.expireSecond = expireSecond
	}
}

// GetQueue 获取组
//
// groupID 组号
func (g *Groups) GetQueue(groupID int64) *Queue {
	g.Lock()
	defer g.Unlock()
	oneQueue, exist := g.queues[groupID]
	if !exist {
		oneQueue = &Queue{
			groupID:    groupID,
			ticketChan: make(chan Ticket, g.queueMax),
			status:     true,
		}
		go g.manager(oneQueue)
		g.queues[groupID] = oneQueue
	}
	return oneQueue
}

// QueueUp 排队取号
func (q *Queue) QueueUp() *Ticket {
	queueChan := make(Ticket)
	q.ticketChan <- queueChan
	return &queueChan
}

func (g *Groups) remove(groupID int64) {
	g.Lock()
	defer g.Unlock()
	oneQueue, exist := g.queues[groupID]
	if exist {
		oneQueue.status = false
		delete(g.queues, groupID)
	}
}

// 队列管理
func (g *Groups) manager(q *Queue) {
	count := int64(0)
	expireTimes := g.expireSecond * 100
	timeoutDuration := time.Duration(g.timeout) * time.Millisecond
	for q.status {
		if q.currentTicket == nil {
			timeout := time.After(10 * time.Millisecond)
			select {
			case currentTicket := <-q.ticketChan:
				count = 0
				q.currentTicket = currentTicket
			case <-timeout:
				if expireTimes > 0 {
					count++
					if count >= expireTimes {
						g.remove(q.groupID)
					}
				}
				break
			}
		}
		if q.currentTicket != nil {
			q.callTicket()
			if timeoutDuration > 0 {
				timeout := time.After(timeoutDuration)
				select {
				case <-timeout:
				case <-q.currentTicket:
				}
			} else {
				<-q.currentTicket
			}
			q.closeTicket()
		}
	}
}

func (q *Queue) callTicket() {
	q.currentTicket <- true
}

func (q *Queue) closeTicket() {
	close(q.currentTicket)
	q.currentTicket = nil
}

// Wait 等待叫号
func (t *Ticket) Wait() {
	<-(*t)
}

// Leave 离开队伍
func (t *Ticket) Leave() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.New("ticket timeout")
		}
	}()
	(*t) <- true
	return
}
