package protocol

import "fmt"

type TestMessage interface {
	//	Protocol
	Type() uint8
}

type TestProtocol interface {
	Encode()
	Decode()
}
type TestMqttProtocol interface {
	TestProtocol
	Read()
	Write()
}

type TestConnect struct {
	clientId string
}

type Test struct {
	retain  uint8
	dupflag uint8
	len     uint64
	off     uint32
}

func (c *TestConnect) Type() uint8 {
	return 1
}

/*func (c *Connect) Encode() {
	fmt.Println("connect")
}

func (c *Connect) Decode() {
	fmt.Println("decode")
}
*/
func (c *TestConnect) Write() {
	fmt.Println("write")
}

func (c *TestConnect) Read() {
	fmt.Println("read")
}

/*func main() {
	NewMsg().Type()
}*/

func NewMsg() TestMessage {
	return new(TestConnect)
}
