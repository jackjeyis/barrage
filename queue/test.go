package queue

import (
	"fmt"
	"time"
)

func main() {
	var i int
L:
	for {
		/*if i == 3 {
			break L
		}
		*/
		time.Sleep(3 * time.Second)
		i++
		fmt.Println(i)
		break L
	}
	fmt.Println("L")
}
