package main

import (
	"fmt"
	"goio/logger"
	"goio/timer"
	"time"
)

func main() {
	logger.Start(logger.EveryHour, logger.PrintStack)
	timer := timer.NewTimer(1, 1000*60*60*24)
	timer_id := timer.AddTimer(1000, func() {
		fmt.Println("hello timer")
	})
	//timer.CancelTimer(timer_id)
	go func() {
		for {
			n := timer.ProcessTimer()
			if n != nil {
				if n.Active() {
					d, ok := n.Data().(func())
					if ok {
						d()
						n = n.Next()
					}
				}
			}
		}
	}()
	fmt.Println(timer_id)
	fmt.Println(timer.AddTimer(3000, func() {
		fmt.Println("how are you")
	}))
	time.Sleep(60 * time.Second)
}
