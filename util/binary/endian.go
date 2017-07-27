package binary

//import "barrage/queue"

var BigEndian bigEndian

type bigEndian struct{}

func (bigEndian) Int16(b []byte) int16 {
	return int16(b[1]) | int16(b[0])<<8
}

/*func (bigEndian) PutInt16(b *queue.ByteBuffer, v int16) {
	b.WriteByte(byte(v >> 8))
	b.WriteByte(byte(v))
}*/

func (bigEndian) Int32(b []byte) int32 {
	return int32(b[3]) | int32(b[2])<<8 | int32(b[1])<<16 | int32(b[0])<<24
}

/*func (bigEndian) PutInt32(b *queue.ByteBuffer, v int32) {
	b.WriteByte(byte(v >> 24))
	b.WriteByte(byte(v >> 16))
	b.WriteByte(byte(v >> 8))
	b.WriteByte(byte(v))
}
*/

func Put(packLen,op,seq int32,ver,hsize int16) (b [16]byte) {
	b[0] = byte(packLen>>24)
	b[1] = byte(packLen>>16)
	b[2] = byte(packLen>>8)
	b[3] = byte(packLen)
	b[4] = byte(hsize>>8)
	b[5] = byte(hsize)
	b[6] = byte(ver>>8)
	b[7] = byte(ver)
	b[8] = byte(op>>24)
	b[9] = byte(op>>16)
	b[10] = byte(op>>8)
	b[11] = byte(op)
	b[12] = byte(seq>>24)
	b[13] = byte(seq>>16)
	b[14] = byte(seq>>8)
	b[15] = byte(seq)
	return
}

func (bigEndian) PutInt32(b [4]byte, v int32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

func (bigEndian) PutInt64(v int64) (b []byte) {
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
	return
}

func (bigEndian) Int64(b []byte) int64 {
	return int64(b[7]) | int64(b[6])<<8 | int64(b[5])<<16 | int64(b[4])<<24 | int64(b[3])<<32 | int64(b[2])<<40 | int64(b[1])<<48 | int64(b[0])<<56
}
