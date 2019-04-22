package groupqueue

import (
	"sync"
	"time"
)

type queueGroup struct {
	Timeout      int64 // 单个号业务办理超时时间（毫秒）
	ExpireSecond int64 // 组队列没有排号后多长时间关闭队列（秒）
	queues       map[int64]*queue
	sync.Mutex
}

type queue struct {
	groupID     int64          // 组ID(窗口ID)
	reqChan     chan chan bool // 排号队列
	currentChan chan bool      // 当前办理号
	status      bool           // 队列状态(true:启用；false:关闭)
}

// QueueGroup 数据请求队列
var QueueGroup *queueGroup

func init() {
	QueueGroup = &queueGroup{
		queues: make(map[int64]*queue),
	}
}

// QueueUp 排队取号
func (g *queueGroup) QueueUp(groupID int64) chan bool {
	queueChan := make(chan bool)
	oneQueue, exist := g.queues[groupID]
	if !exist {
		g.Lock()
		oneQueue, exist = g.queues[groupID]
		if !exist {
			oneQueue = &queue{
				groupID: groupID,
				reqChan: make(chan chan bool, 10),
				status:  true,
			}
			go oneQueue.manager()
			g.queues[groupID] = oneQueue
		}
		g.Unlock()
	}

	oneQueue.reqChan <- queueChan
	return queueChan
}

func (g *queueGroup) remove(groupID int64) {
	g.Lock()
	defer g.Unlock()
	oneQueue, exist := g.queues[groupID]
	if exist {
		oneQueue.status = false
		delete(g.queues, groupID)
	}
}

// 队列管理
func (q *queue) manager() {
	count := int64(0)
	expireTimes := QueueGroup.ExpireSecond * 100
	timeoutDuration := time.Duration(QueueGroup.Timeout) * time.Millisecond
	for q.status {
		if q.currentChan == nil {
			timeout := time.After(10 * time.Millisecond)
			select {
			case currentChan := <-q.reqChan:
				count = 0
				q.currentChan = currentChan
				q.currentChan <- true
			case <-timeout:
				if expireTimes > 0 {
					count++
					if count >= expireTimes {
						QueueGroup.remove(q.groupID)
					}
				}
				break
			}
		}
		if q.currentChan != nil {
			if timeoutDuration > 0 {
				timeout := time.After(timeoutDuration)
				select {
				case <-timeout:
					q.currentChan = nil
				case <-q.currentChan:
					q.currentChan = nil
				}
			} else {
				<-q.currentChan
				q.currentChan = nil
			}

		}
	}
}
