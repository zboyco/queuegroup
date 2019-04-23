package main

import (
	"fmt"
	"sync"

	"github.com/zboyco/queuegroup"
)

func main() {
	wg := sync.WaitGroup{}
	// 配置超时和过期时间
	queuegroup.Config(
		100, // 单个队列最大长度（默认10）
		500, // 单个号业务办理超时时间（毫秒，默认不超时）
		30,  // 组队列没有排号后多长时间关闭队列（秒，默认不关闭）
	)
	// 获取队列
	queue := [2]*queuegroup.Queue{
		queuegroup.GetQueue(0),
		queuegroup.GetQueue(1),
	}

	data := [2]int{
		0,
		0,
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)

		groupID := i % 2
		// 取号
		ticket := queue[groupID].QueueUp()

		go func(mt *queuegroup.Ticket, id int, g int) {
			// 等待叫号
			mt.Wait()

			// 办理业务
			data[g]++
			fmt.Printf("[%v组%v号]办理成功: %v \n", g, id, data[g])

			// 离开队伍
			err := mt.Leave()
			if err != nil {
				fmt.Println(err)
			}

			wg.Done()
		}(ticket, i, groupID)
	}
	wg.Wait()
}
