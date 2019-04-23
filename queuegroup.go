package queuegroup

import (
	"errors"
	"sync"
	"time"
)

// Groups 队列组
type Groups struct {
	Timeout      int64 // 单个号业务办理超时时间（毫秒）
	ExpireSecond int64 // 组队列没有排号后多长时间关闭队列（秒）
	queues       map[int64]*queue
	sync.Mutex
}

// Ticket 排队票据
type Ticket chan bool

type queue struct {
	groupID       int64       // 组ID(窗口ID)
	ticketChan    chan Ticket // 排号队列
	currentTicket Ticket      // 当前办理号
	status        bool        // 队列状态(true:启用；false:关闭)
}

// QueueGroup 数据请求队列
var QueueGroup *Groups

func init() {
	QueueGroup = &Groups{
		queues: make(map[int64]*queue),
	}
}

// NewQueueGroup 新建一个队列组
func NewQueueGroup() *Groups {
	return &Groups{
		queues: make(map[int64]*queue),
	}
}

// QueueUp 排队取号
func (g *Groups) QueueUp(groupID int64) *Ticket {
	queueChan := make(Ticket, 1)
	oneQueue, exist := g.queues[groupID]
	if !exist {
		g.Lock()
		oneQueue, exist = g.queues[groupID]
		if !exist {
			oneQueue = &queue{
				groupID:    groupID,
				ticketChan: make(chan Ticket, 10),
				status:     true,
			}
			go g.manager(oneQueue)
			g.queues[groupID] = oneQueue
		}
		g.Unlock()
	}

	oneQueue.ticketChan <- queueChan
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
func (g *Groups) manager(q *queue) {
	count := int64(0)
	expireTimes := g.ExpireSecond * 100
	timeoutDuration := time.Duration(g.Timeout) * time.Millisecond
	for q.status {
		if q.currentTicket == nil {
			timeout := time.After(10 * time.Millisecond)
			select {
			case currentTicket := <-q.ticketChan:
				count = 0
				q.currentTicket = currentTicket
				q.currentTicket <- true
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
			if timeoutDuration > 0 {
				timeout := time.After(timeoutDuration)
				select {
				case <-timeout:
					close(q.currentTicket)
					q.currentTicket = nil
				case <-q.currentTicket:
					close(q.currentTicket)
					q.currentTicket = nil
				}
			} else {
				<-q.currentTicket
				q.currentTicket = nil
			}

		}
	}
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
