package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

func main() {

	var (
		c     *zk.Conn
		paths []string
		data  []byte
		err   error
	)
	c, _, err = zk.Connect([]string{"172.16.6.74"}, 500*time.Millisecond)
	if err != nil {
		fmt.Println(err)
	}
	http.HandleFunc("/get/ip", func(w http.ResponseWriter, r *http.Request) {
		paths, _, err = c.Children("/barrage")
		if len(paths) != 0 {
			data, _, err = c.Get("/barrage/" + paths[0])
			if err != nil {
				fmt.Println("Get", err)
			}
			fmt.Println(string(data))
		}
		fmt.Fprint(w, string(data))
	})

	http.ListenAndServe(":8888", nil)
}
