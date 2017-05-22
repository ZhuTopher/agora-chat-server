package main

import(
	"bytes"
	"encoding/binary"
)

// Message types
const (
	// Server client auth request
	MTypeClientAuth uint8 = iota
	// Client auth responses
	MTypeClientID
	MTypeClientName
	// Client TCP text message
	MTypeClientText
)

type Message struct {
	Type 		uint8
	Data 		interface{}
}

type MsgClientText struct {
	ClientID	uint32
	TextBytes	[]byte
	// TODO: actually parse TextBytes into a string
	//       in order to perform word checks, etc.
}

func (msg Message) TypeToString() string {
	switch msg.Type {
	case MTypeClientAuth:
		return "MTypeClientAuth"
	case MTypeClientID:
		return "MTypeClientID"
	case MTypeClientName:
		return "MTypeClientName"
	case MTypeClientText:
		return "MTypeClientText"
	}
	panic("TypeToString(): unknown Message type.")
}

// WRITING
// returns the raw binary bit data of msg.Data
func (msg *Message) ToBinary() (msgBytes []byte, err error) {
	buf := new(bytes.Buffer)

	// 1st: write message type to buf
	err = binary.Write(buf, binary.BigEndian, msg.Type)
	if err != nil {
		return nil, err
	}

	// 2nd: write message data w/ specific format to buf
	// note: first 32 bits (uint32) should be length of remaining msg
	bin, err := msg.handleDataToBinary()
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(bin)
	if err != nil {
		return nil, err
	}

	msgBytes = buf.Bytes()
	return // msgBytes, nil
}

// first 32 bits (uint32) of buf should ALWAYS be the length of the
// following (binary-encoded) message
func (msg *Message) handleDataToBinary() (bin []byte, err error) {
	switch msg.Data.(type) {
	case MsgClientText:
		bin, err = dataToBinary(msg.Data.(MsgClientText))
	default:
		panic("handleDataToBinary(): unknown Message type.")
	}
	if err != nil {
		return nil, err
	}

	return // bin, nil
}

// bit pattern: 32, 16, len(data.TextBytes)
//    - 32: length of following message
//    - 32: ClientID (uint16)
//    - len(data.TextBytes): data.TextBytes
func dataToBinary(data MsgClientText) (dataBytes []byte, err error) {
	buf := new(bytes.Buffer)
	dataBuf := new(bytes.Buffer)

	err = binary.Write(dataBuf, binary.BigEndian, data.ClientID)
	if err != nil {
		return nil, err
	}
	_, err = dataBuf.Write(data.TextBytes)
	if err != nil {
		return nil, err
	}

	bytes := dataBuf.Bytes()
	err = binary.Write(buf, binary.BigEndian, uint32(len(bytes)))
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(bytes)
	if err != nil {
		return nil, err
	}

	dataBytes = buf.Bytes()
	return // dataBytes, nil
}

// READING
// bin is the binary encoding of the Message.Data
// (i.e bin does not contain bit data for msgType NOR the length)
func MsgFromBinary(msgType uint8, bin []byte) (msg *Message, err error) {
	var data interface{}
	switch msgType {
	case MTypeClientText:
		data, err = NewMsgClientText(bin)
	default:
		panic("MsgFromBinary(): unknown Message type.")
	}
	if err != nil {
		return nil, err
	}

	return &Message{msgType, data}, nil
}

func NewMsgClientText(bin []byte) (data *MsgClientText, err error) {
	var (
		cID 		uint32
		textBytes 	[]byte
	)

	buf := bytes.NewReader(bin)
	err = binary.Read(buf, binary.BigEndian, cID)
	if err != nil {
		return nil, err
	}
	// TODO: Non-hardcoded value, 4*8 = 32
	textBytes = bin[8:]
	/*textBytes := make([]byte, len(bin)-8)
	_, err = io.ReadFull(c.connReader, msgData)
	if err != nil {
		return nil, err
	}*/

	return &MsgClientText{cID, textBytes}, nil
}
