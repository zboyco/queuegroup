package queuegroup_test

import (
	"sync"
	"testing"

	"github.com/zboyco/queuegroup"
)

func Test_QueueUp(t *testing.T) {

	// 新建队列组
	dataQueue := queuegroup.NewQueueGroup()
	// 配置超时和过期时间
	dataQueue.Config(10, 500, 30)
	// 获取队列
	queue := dataQueue.GetQueue(1)

	// 取号
	ticket := queue.QueueUp()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func(mt *queuegroup.Ticket) {
		// 等待叫号
		mt.Wait()
		t.Log("办理成功")
		// 离开队伍
		mt.Leave()
		wg.Done()
	}(ticket)

	wg.Wait()
}
