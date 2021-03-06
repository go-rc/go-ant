// This is a marshallable datastructure for constructing and decoding ant packets
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	maxDataLength = 56
	syncByte      = 0xA4
)

type antpacketTemplate struct {
	datalength byte
	id         byte
}

// Encapsulates the basic structure of a standard ant packet.
//
// Byte #	-	Label
// 0	-	Sync
// 1	-	Message Length
// 2	- Message ID
// 3-(Message Length+2) - Data bytes (LSB ORDERING!)
// (Message Length + 3) - Checksum (XOR of all previous bytes including SYNC)
type antpacket struct {
	sync     byte
	msglen   byte
	id       byte
	data     []byte
	checksum byte
}

func GenerateAntpacket(class byte, args ...byte) (*antpacket, error) {
	if args == nil {
		return nil, ErrArgumentsNil
	}

	v, ok := msgClasses[class]
	if ok == false {
		return nil, ErrUnknownClass
	}

	if len(args) != int(v.template.datalength) {
		return nil, ErrArgumentsLen
	}

	pkt := &antpacket{
		syncByte,
		v.template.datalength,
		v.template.id,
		make([]byte, v.template.datalength),
		0,
	}

	for i, x := range args {
		pkt.data[i] = x
	}

	pkt.setChecksum()

	return pkt, nil
}

// Stringify the packet with field explainations
func (a *antpacket) String() string {
	// Get msgClass
	c := msgClasses[a.id]

	head := fmt.Sprint(
		"\"", c.name, "\" - ",
		"Class: ", c.class, "\n\tData: ",
	)

	data := ""

	for i, x := range c.dataFieldDesc {
		data = fmt.Sprintf("%s%s - %X, ",
			data,
			x,
			a.data[i],
		)
	}

	return fmt.Sprintln(head, data)
}

// Checksum the packet
func (a *antpacket) genChecksum() (chk byte) {
	// XOR everything
	chk = a.sync ^ a.msglen ^ a.id
	for _, e := range a.data {
		chk = chk ^ e
	}
	return
}

// Set the checksum for a constructed packet
func (a *antpacket) setChecksum() {
	a.checksum = a.genChecksum()
}

// Validate a packet by checksum
// Returns true if valid
func (a *antpacket) validateChecksum() bool {
	if a.genChecksum() == a.checksum {
		return true
	}
	return false
}

// Encode to line format
func (a *antpacket) toBinary(buffer *bytes.Buffer) (length int, err error) {
	// TODO: more elegant than this
	binary.Write(buffer, binary.LittleEndian, a.sync)
	binary.Write(buffer, binary.LittleEndian, a.msglen)
	binary.Write(buffer, binary.LittleEndian, a.id)
	binary.Write(buffer, binary.LittleEndian, a.data)
	binary.Write(buffer, binary.LittleEndian, a.checksum)

	length = buffer.Len()
	return
}

// Unpack from line format
func readAntpacket(buf []byte) (*antpacket, error) {
	// Minimum Length check
	if len(buf) < 5 {
		return nil, ErrMinimumPacketLength
	}

	ret := &antpacket{}
	stream := bytes.NewReader(buf)

	ret.sync, _ = stream.ReadByte()
	ret.msglen, _ = stream.ReadByte()
	ret.id, _ = stream.ReadByte()
	data := make([]byte, ret.msglen)
	_, err := stream.Read(data)
	if err != nil {
		return nil, err
	}
	ret.data = data
	ret.checksum, _ = stream.ReadByte()

	// Verify checksum
	if ret.genChecksum() != ret.checksum {
		return nil, ErrChecksumMismatch
	}

	return ret, nil
}
