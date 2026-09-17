package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

// READER

/*
RESP type prefix bytes.

Redis Serialization Protocol (RESP) uses the first byte of each value
to identify the type of data that follows.
*/
const (
	RESPstring  = '+'
	RESPerror   = '-'
	RESPinteger = ':'
	RESPbulk    = '$'
	RESParray   = '*'
)

/*
This is the internal representation when input is recieved in RESP form

Instead of keeping RESP values as raw bytes, incoming data is parsed
into this structure so command handlers can work with normal Go values.

For example, the RESP bulk string;

	$4\r\nfoo\r\n

becomes;

	Value{
		typ:  "bulk",
		bulk: "foo",
	}

Different fields are used depending on the RESP value type;

- str   stores simple strings and errors
- num   stores RESP integers
- bulk  stores bulk strings
- array stores nested RESP values
- hash  stores Redis hash data used internally by the database
*/
type Value struct {
	typ   string
	str   string
	num   int
	bulk  string
	array []Value
	hash  map[string]string
}

type Resp struct {
	reader *bufio.Reader
}

func NewResp(rd io.Reader) *Resp {
	return &Resp{reader: bufio.NewReader(rd)}
}

// reads byte until it reaches \r, return without \r\n
func (r *Resp) readline() (line []byte, n int, err error) {
	for {
		b, err := r.reader.ReadByte()

		if err != nil {
			return nil, 0, err
		}

		n += 1
		line = append(line, b)

		if len(line) >= 2 && line[len(line)-2] == '\r' {
			break
		}
	}

	return line[:len(line)-2], n, nil
}

// read integer to get the size of the following value
func (r *Resp) readinteger() (x int, y int, err error) {
	line, n, err := r.readline()

	if err != nil {
		return 0, 0, err
	}

	// convert bytes in line into a 64 bits signed integer e.g. "123"
	// parse it as base 10 (decimal) and as a signed 64 bit integer
	i64, err := strconv.ParseInt(string(line), 10, 64)

	if err != nil {
		return 0, n, err
	}

	return int(i64), n, nil
}

/*
read() reads a single RESP value from the input stream.
RESP values begin with a one-byte prefix describing their type.
*/
func (r *Resp) read() (Value, error) {
	_type, err := r.reader.ReadByte()

	if err != nil {
		return Value{}, err
	}

	switch _type {
	case RESParray:
		return r.readarray()
	case RESPbulk:
		return r.readbulk()
	default:
		fmt.Printf("unknown type: %v", string(_type))
		return Value{}, nil
	}
}

func (r *Resp) readarray() (Value, error) {
	v := Value{}
	v.typ = "array"

	length, _, err := r.readinteger()

	if err != nil {
		return v, err
	}

	v.array = make([]Value, 0)

	for i := 0; i < length; i++ {
		val, err := r.read()

		if err != nil {
			return v, err
		}

		v.array = append(v.array, val)
	}

	return v, nil
}

func (r *Resp) readbulk() (Value, error) {
	v := Value{}
	v.typ = "bulk"

	length, _, err := r.readinteger()

	if err != nil {
		return v, err
	}

	bulk := make([]byte, length)

	_, err = io.ReadFull(r.reader, bulk)
	v.bulk = string(bulk)

	// read trailing crlf
	r.readline()

	return v, nil
}

// WRITER

/*
Values are first converted from the server's internal Value representation
into RESP bytes before being written to the client.
*/
type writer struct {
	writer io.Writer
}

func NewWriter(w io.Writer) *writer {
	return &writer{writer: w}
}

/*
write serializes a Value into RESP format and sends it to the client.
The Value itself does not contain raw RESP bytes. Instead, marshall()
determines the appropriate encoding according to val.typ.
*/
func (w *writer) write(val Value) error {
	var bytes = val.marshall()

	_, err := w.writer.Write(bytes)

	if err != nil {
		return err
	}

	return nil
}

func (val Value) marshall() []byte {
	switch val.typ {
	case "array":
		return val.marshallarray()
	case "bulk":
		return val.marshallbulk()
	case "string":
		return val.marshallstring()
	case "number":
		return val.marshallnumber()
	case "null":
		return val.marshallnull()
	case "error":
		return val.marshallerror()
	default:
		return []byte{}
	}
}

func (val Value) marshallstring() []byte {
	var bytes []byte

	bytes = append(bytes, RESPstring)
	bytes = append(bytes, val.str...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (val Value) marshallbulk() []byte {
	var bytes []byte

	bytes = append(bytes, RESPbulk)
	bytes = append(bytes, strconv.Itoa(len(val.bulk))...)
	bytes = append(bytes, '\r', '\n')
	bytes = append(bytes, val.bulk...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (val Value) marshallarray() []byte {
	var bytes []byte
	length := len(val.array)

	bytes = append(bytes, RESParray)
	bytes = append(bytes, strconv.Itoa(length)...)
	bytes = append(bytes, '\r', '\n')

	for i := 0; i < length; i++ {
		bytes = append(bytes, val.array[i].marshall()...)
	}

	return bytes
}

func (val Value) marshallnumber() []byte {
	var bytes []byte

	bytes = append(bytes, RESPinteger)
	bytes = append(bytes, strconv.Itoa(val.num)...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (val Value) marshallerror() []byte {
	var bytes []byte

	bytes = append(bytes, RESPerror)
	bytes = append(bytes, val.str...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (val Value) marshallnull() []byte {
	var bytes []byte

	bytes = append(bytes, "$-1\r\n"...)

	return bytes
}
