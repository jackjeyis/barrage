package protocol

import (
	"errors"

	"barrage/logger"
	"barrage/msg"
	"barrage/queue"
	"sync"
)

const (
	MaxPayloadSize = (1 << (4 * 7)) - 1
)

var (
	badMsgTypeErr    = errors.New("mqtt: message type is invalid")
	badQosErr        = errors.New("mqtt: Qos is invalid")
	badWillQosErr    = errors.New("mqtt: Will Qos is invalid")
	badLenErr        = errors.New("mqtt: remaining length field exceeded maximum of 4 bytes")
	badReturnCodeErr = errors.New("mqtt: return code is invalid")
	badPacketErr     = errors.New("mqtt: data exceeds packet length")
	badMsgLenErr     = errors.New("mqtt: message is too long")
	HeaderErr        = errors.New("mqtt: header bytes not enough")
	BodyErr          = errors.New("mqtt: body bytes not enough")
)

type Qoslevel uint8

type MsgType uint8

type ReturnCode uint8

const (
	QosAtMostOnce = Qoslevel(iota)
	QosAtLeastOnce
	QosExactlyOnce

	QosInvalid
)

const (
	Connect = MsgType(iota + 1)
	ConnAck
	Publish
	PubAck
	PubRec
	PubRel
	PubComp
	Subscribe
	SubAck
	UnSub
	UnSubAck
	PingReq
	PingRes
	DisConnect

	MsgTypeInvalid
)

const (
	RetCodeAccepted = ReturnCode(iota)
	RetCodeInvalidVersion
	RetCodeIdRejected
	RetCodeServerUnavailable
	RetCodeUsernameOrPwd
	RetCodeUnAuthorized
)

func (qos Qoslevel) IsValid() bool {
	return qos >= QosAtMostOnce && qos < QosInvalid
}

func (qos Qoslevel) IsMsgId() bool {
	return qos == QosAtMostOnce || qos == QosAtLeastOnce
}

func (msgType MsgType) IsValid() bool {
	return msgType > Connect && msgType < MsgTypeInvalid
}

func (retCode ReturnCode) IsValid() bool {
	return retCode >= RetCodeAccepted && retCode < 255
}

type Header struct {
	msgType  MsgType
	retain   bool
	dupflag  bool
	qosLevel Qoslevel
}

func (h *Header) Decode(b *queue.IOBuffer) (remainLen int32, err error) {
	temp := b.Read(uint64(1))
	*h = Header{
		msgType:  MsgType(temp[0] & 0xF0 >> 4),
		retain:   temp[0]&0x01 > 0,
		dupflag:  temp[0]&0x08 > 0,
		qosLevel: Qoslevel(temp[0] & 0x06 >> 1),
	}
	remainLen = decodeLength(b)
	return
}

func (h *Header) Encode(b *queue.ByteBuffer, remainLen int32) error {
	if !h.qosLevel.IsValid() {
		return badQosErr
	}

	if !h.msgType.IsValid() {
		return badMsgTypeErr
	}

	flag := byte(h.msgType) << 4
	flag |= boolToByte(h.dupflag) << 3
	flag |= byte(h.qosLevel) << 1
	flag |= boolToByte(h.retain)
	b.WriteByte(flag)
	encodeLength(b, remainLen)
	return nil
}

func writeHeader(b *queue.IOBuffer, h *Header, variableHeader *queue.ByteBuffer, payloadSize int32) error {
	totalLen := int32(int32(variableHeader.Len()) + payloadSize)
	if totalLen > MaxPayloadSize {
		return badMsgLenErr
	}
	buf := queue.Get()
	err := h.Encode(buf, totalLen)
	if err != nil {
		return err
	}
	err = b.Write(buf.Bytes())
	if err != nil {
		return err
	}

	err = b.Write(variableHeader.Bytes())
	if err != nil {
		return err
	}

	queue.Put(buf)
	queue.Put(variableHeader)
	return nil
}

type MqttConnect struct {
	id              int
	channel         msg.Channel
	ProtocolName    string
	ProtocolVersion uint8
	CleanSession    bool
	WillFlag        bool
	WillQos         Qoslevel
	WillRetain      bool
	PasswordFlag    bool
	UserNameFlag    bool
	KeepAlive       uint16
	ClientId        string
	WillTopic       string
	WillMessage     string
	UserName        string
	Password        string
}

func (c *MqttConnect) Type() uint8 {
	return uint8(Connect)
}

func (c *MqttConnect) SetChannel(ch msg.Channel) {
	c.channel = ch
}

func (c *MqttConnect) Channel() msg.Channel {
	return c.channel
}

func (c *MqttConnect) SetHandlerId(i int) {
	c.id = i
}

func (c *MqttConnect) HandlerId() int {
	return c.id
}

func (c *MqttConnect) Decode(b *queue.IOBuffer, remainLen int32) (err error) {
	c.ProtocolName = getString(b, &remainLen)
	c.ProtocolVersion = getUint8(b, &remainLen)
	flag := getUint8(b, &remainLen)
	c.UserNameFlag = flag&0x80 > 0
	c.PasswordFlag = flag&0x40 > 0
	c.WillRetain = flag&0x20 > 0
	c.WillQos = Qoslevel(flag & 0x18 >> 3)
	c.WillFlag = flag&0x04 > 0
	c.CleanSession = flag&0x02 > 0
	c.KeepAlive = getUint16(b, &remainLen)
	c.ClientId = getString(b, &remainLen)

	if c.WillFlag {
		c.WillTopic = getString(b, &remainLen)
		c.WillMessage = getString(b, &remainLen)
	}

	if c.UserNameFlag && c.PasswordFlag {
		c.UserName = getString(b, &remainLen)
		c.Password = getString(b, &remainLen)
	}
	if remainLen != 0 {
		return badMsgLenErr
	}
	return nil
}

func (c *MqttConnect) Encode(b *queue.IOBuffer) error {
	buf := queue.Get()
	if !c.WillQos.IsValid() {
		return badWillQosErr
	}
	flag := boolToByte(c.UserNameFlag) << 7
	flag |= boolToByte(c.PasswordFlag) << 6
	flag |= boolToByte(c.WillRetain) << 5
	flag |= byte(c.WillQos) << 3
	flag |= boolToByte(c.CleanSession) << 1
	setString(c.ProtocolName, buf)
	setUint8(c.ProtocolVersion, buf)
	setUint8(flag, buf)
	setUint16(c.KeepAlive, buf)
	setString(c.ClientId, buf)
	if c.WillFlag {
		setString(c.WillTopic, buf)
		setString(c.WillMessage, buf)
	}

	if c.UserNameFlag && c.PasswordFlag {
		setString(c.UserName, buf)
		setString(c.Password, buf)
	}
	h := &Header{
		msgType: Connect,
	}
	return writeHeader(b, h, buf, 0)
}

type MqttConnAck struct {
	id      int
	channel msg.Channel
	RetCode ReturnCode
}

func (c *MqttConnAck) Type() uint8 {
	return uint8(ConnAck)
}

func (c *MqttConnAck) SetChannel(ch msg.Channel) {
	c.channel = ch
}

func (c *MqttConnAck) Channel() msg.Channel {
	return c.channel
}

func (c *MqttConnAck) SetHandlerId(i int) {
	c.id = i
}

func (c *MqttConnAck) HandlerId() int {
	return c.id
}

func (c *MqttConnAck) Encode(b *queue.IOBuffer) error {
	buf := queue.Get()
	buf.WriteByte(byte(0))
	setUint8(uint8(c.RetCode), buf)
	h := &Header{
		msgType: ConnAck,
	}
	return writeHeader(b, h, buf, 0)
}

func (c *MqttConnAck) Decode(b *queue.IOBuffer, remainLen int32) error {
	getUint8(b, &remainLen)
	c.RetCode = ReturnCode(getUint8(b, &remainLen))
	if !c.RetCode.IsValid() {
		return badReturnCodeErr
	}

	if remainLen != 0 {
		return badMsgLenErr
	}
	return nil
}

type MqttPublish struct {
	id      int
	channel msg.Channel
	Header
	Topic   []byte
	MsgId   uint16
	Payload []byte
}

func (p *MqttPublish) Type() uint8 {
	return uint8(Publish)
}

func (p *MqttPublish) SetChannel(ch msg.Channel) {
	p.channel = ch
}

func (p *MqttPublish) Channel() msg.Channel {
	return p.channel
}

func (p *MqttPublish) SetHandlerId(i int) {
	p.id = i
}

func (p *MqttPublish) HandlerId() int {
	return p.id
}

var mutex sync.Mutex

func (p *MqttPublish) Encode(b *queue.IOBuffer) error {
	mutex.Lock()
	mutex.Unlock()
	buf := queue.Get()
	setBytes(p.Topic, buf)
	if p.Header.qosLevel.IsMsgId() {
		setUint16(p.MsgId, buf)
	}
	writeHeader(b, &p.Header, buf, int32(len(p.Payload)))
	b.Write(p.Payload)
	return nil
}

func (p *MqttPublish) Decode(b *queue.IOBuffer, remainLen int32) error {
	p.Topic = getBytes(b, &remainLen)

	if p.Header.qosLevel.IsMsgId() {
		p.MsgId = getUint16(b, &remainLen)
	}

	p.Payload = getPayload(b, &remainLen)

	if remainLen != 0 {
		return badMsgLenErr
	}
	return nil
}

type MqttPubAck struct {
	id      int
	channel msg.Channel
	MsgId   uint16
}

func (p *MqttPubAck) Type() uint8 {
	return uint8(PubAck)
}

func (p *MqttPubAck) SetChannel(ch msg.Channel) {
	p.channel = ch
}

func (p *MqttPubAck) Channel() msg.Channel {
	return p.channel
}

func (p *MqttPubAck) SetHandlerId(i int) {
	p.id = i
}

func (p *MqttPubAck) HandlerId() int {
	return p.id
}

func (p *MqttPubAck) Encode(b *queue.IOBuffer) error {
	buf := queue.Get()
	setUint16(p.MsgId, buf)
	h := &Header{
		msgType: PubAck,
	}
	return writeHeader(b, h, buf, 0)
}

func (p *MqttPubAck) Decode(b *queue.IOBuffer, remainLen int32) error {
	p.MsgId = getUint16(b, &remainLen)

	if remainLen != 0 {
		return badMsgLenErr
	}

	return nil
}

type MqttPingReq struct {
	id      int
	channel msg.Channel
}

func (p *MqttPingReq) Type() uint8 {
	return uint8(PingReq)
}

func (p *MqttPingReq) SetChannel(ch msg.Channel) {
	p.channel = ch
}

func (p *MqttPingReq) Channel() msg.Channel {
	return p.channel
}

func (p *MqttPingReq) SetHandlerId(i int) {
	p.id = i
}

func (p *MqttPingReq) HandlerId() int {
	return p.id
}

func (p *MqttPingReq) Encode(b *queue.IOBuffer) error {
	buf := queue.Get()
	h := &Header{
		msgType: PingReq,
	}
	return writeHeader(b, h, buf, 0)
}

func (p *MqttPingReq) Decode(b *queue.IOBuffer, remainLen int32) error {
	if remainLen != 0 {
		return badMsgLenErr
	}
	return nil
}

type MqttPingRes struct {
	id      int
	channel msg.Channel
}

func (p *MqttPingRes) Type() uint8 {
	return uint8(PingRes)
}

func (p *MqttPingRes) SetChannel(ch msg.Channel) {
	p.channel = ch
}

func (p *MqttPingRes) Channel() msg.Channel {
	return p.channel
}

func (p *MqttPingRes) SetHandlerId(i int) {
	p.id = i
}

func (p *MqttPingRes) HandlerId() int {
	return p.id
}

func (p *MqttPingRes) Encode(b *queue.IOBuffer) error {
	buf := queue.Get()
	h := &Header{
		msgType: PingRes,
	}
	return writeHeader(b, h, buf, 0)
}

func (p *MqttPingRes) Decode(b *queue.IOBuffer, remainLen int32) error {
	if remainLen != 0 {
		return badMsgLenErr
	}
	return nil
}

type MqttDisConnect struct {
	id      int
	channel msg.Channel
}

func (c *MqttDisConnect) Type() uint8 {
	return uint8(DisConnect)
}

func (c *MqttDisConnect) SetChannel(ch msg.Channel) {
	c.channel = ch
}

func (c *MqttDisConnect) Channel() msg.Channel {
	return c.channel
}

func (c *MqttDisConnect) SetHandlerId(i int) {
	c.id = i
}

func (c *MqttDisConnect) HandlerId() int {
	return c.id
}

func (c *MqttDisConnect) Encode(b *queue.IOBuffer) error {
	buf := queue.Get()
	h := &Header{
		msgType: DisConnect,
	}
	return writeHeader(b, h, buf, 0)
}

func (c *MqttDisConnect) Decode(b *queue.IOBuffer, remainLen int32) error {
	if remainLen != 0 {
		return badMsgLenErr
	}
	return nil
}

type MqttProtocol struct {
	Header
	remainLen int32
}

func (m *MqttProtocol) Encode(msg msg.Message, buf *queue.IOBuffer) error {
	return msg.Encode(buf)
}

func (m *MqttProtocol) Decode(buf *queue.IOBuffer) (msg.Message, error) {
	var (
		cnt       int = 2
		remainLen int32
		err       error
	)
	if buf.Init() {
		for {
			if cnt > 5 {
				return nil, errors.New("extend header size")
			}
			if buf.GetReadSize() < uint64(cnt+1) {
				return nil, HeaderErr
			}

			if buf.Byte(uint64(cnt)) >= 0x80 {
				cnt++
			} else {
				break
			}
		}

	}

	if buf.Init() {
		remainLen, err = m.Header.Decode(buf)
		buf.SetInit(false)
	} else {
		remainLen = m.remainLen
	}
	if uint64(remainLen) > buf.GetReadSize() {
		logger.Info("remainLen %v,size  %v", remainLen, buf.GetReadSize())
		m.remainLen = remainLen
		return nil, BodyErr
	}
	if err != nil {
		logger.Error("MqttProtocol.Header.Decode error %v", err)
		return nil, err
	}
	return decodeMessage(m.Header, buf, remainLen)
}

func decodeMessage(header Header, b *queue.IOBuffer, remainLen int32) (msg msg.Message, err error) {
	switch header.msgType {
	case Connect:
		msg = &MqttConnect{}
	case ConnAck:
		msg = &MqttConnAck{}
	case Publish:
		msg = &MqttPublish{Header: header}
	case PubAck:
		msg = &MqttPubAck{}
	case PingReq:
		msg = &MqttPingReq{}
	default:
		return nil, errors.New("Unknown MsgType")
	}
	err = msg.Decode(b, remainLen)
	return
}

func decodeLength(b *queue.IOBuffer) (len int32) {
	var shift uint
	for i := 0; i < 4; i++ {
		temp := b.Read(uint64(1))
		len |= int32(temp[0]&0x7f) << shift
		if temp[0]&0x80 == 0 {
			return
		}
		shift += 7
	}
	return
}

func encodeLength(b *queue.ByteBuffer, len int32) {
	if len == 0 {
		b.WriteByte(byte(0))
	}

	for len > 0 {
		temp := len & 0x7f
		len >>= 7
		if len > 0 {
			temp |= 0x80
		}
		b.WriteByte(byte(temp))
	}
}

func getUint8(b *queue.IOBuffer, remainLen *int32) uint8 {
	if *remainLen < 1 {
		panic(badPacketErr)
	}
	temp := b.Read(uint64(1))
	*remainLen--
	return temp[0]
}

func getUint16(b *queue.IOBuffer, remainLen *int32) uint16 {
	if *remainLen < 2 {
		panic(badPacketErr)
	}
	temp := b.Read(uint64(2))
	*remainLen -= 2
	return uint16(temp[0])<<8 | uint16(temp[1])
}

func getString(b *queue.IOBuffer, remainLen *int32) string {
	return string(getBytes(b, remainLen))
}

func getBytes(b *queue.IOBuffer, remainLen *int32) []byte {
	byteLen := getUint16(b, remainLen)
	temp := b.Read(uint64(byteLen))
	*remainLen -= int32(byteLen)
	return temp
}

func getPayload(b *queue.IOBuffer, remainLen *int32) []byte {
	temp := b.Read(uint64(*remainLen))
	*remainLen = 0
	return temp
}

func setUint8(v uint8, b *queue.ByteBuffer) {
	b.WriteByte(v)
}

func setUint16(v uint16, b *queue.ByteBuffer) {
	b.WriteByte(byte(v & 0xff00 >> 8))
	b.WriteByte(byte(v & 0x00ff))
}

func setString(v string, b *queue.ByteBuffer) {
	b.Write([]byte(v))
}

func setBytes(bs []byte, b *queue.ByteBuffer) {
	l := len(bs)
	setUint16(uint16(l), b)
	b.Write(bs)
}

func boolToByte(b bool) byte {
	if b {
		return byte(1)
	}
	return byte(0)
}
