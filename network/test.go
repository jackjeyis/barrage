package network

import (
	"fmt"
	"sync"
	"time"
)

type Singer interface {
	Sing()
}

type Sayer interface {
	Say()
}

type SayerSayer interface {
	Singer
	Sayer
}

type Person struct {
	name string
}

type Student struct {
	*Person
	age int
}

func (p Person) Say() {
	fmt.Printf("%s can Say\n", p.name)
}

func (p Person) SayPerson() {
	fmt.Printf("%s can SayPerson\n", p.name)
}

func (s Student) Sing() {
	fmt.Printf("%s can Sing\n", s.name)
}

func Test() {
	go OnRead()
	go OnWrite()
}

var mutex sync.Mutex

func OnRead() {
	//mutex.Lock()
	for {
		fmt.Println("sleep 3 seconds read")
		time.Sleep(3 * time.Second)
		fmt.Println("read 3 over")
	}
	//mutex.Unlock()
}
func OnWrite() {
	//mutex.Lock()
	for {
		fmt.Println("sleep 3 seconds write")
		time.Sleep(3 * time.Second)
		fmt.Println("write 3 over")
	}
	//mutex.Unlock()
}
func main() {
	/*var ss SayerSayer = Student{&Person{name: "jay"}, 25} //这里报错不太理解，为什么下面不报错
	//var ss SayerSayer = &Student{Person{"jay"}, 25}
	var s Student = Student{&Person{"jay"}, 23}

	ss.Sing()
	ss.Say()
	s.SayPerson()
	*/
	Test()
	for {
		fmt.Println("test run success")
		time.Sleep(10 * time.Second)
	}
}
