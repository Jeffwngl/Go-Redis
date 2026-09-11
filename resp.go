package main

import (
    "bufio"
    "fmt"
    "io"
    "strconv"
)

// Reader

const (
    STRING = '+'
    ERROR = '-'
    INTEGER = ':'
    BULK = '$'
    ARRAY = '*'
)

// struct to hold all commands we recieve from the client
type Value struct {
    typ string
    str string
    num int
    bulk string
    array []Value
}

// reader to help use read from the buffer and store in the Value struct
type Resp struct {
    reader *bufio.Reader
}
// initialize new Resp object
func NewResp(rd io.Reader) *Resp {
    return &Resp{reader: bufio.NewReader(rd)}
}

// function to read byte until it reaches \r, return without \r\n
func (r *Resp) readLine() (line []byte, n int, err error) {
    for {
        b, err := r.reader.ReadByte()

        if err != nil {
            return nil, 0, err
        }
        
        n += 1;
        line = append(line, b)
        
        if len(line) >= 2 && line[len(line) - 2] == '\r' {
            break
        }
    }
    return line[:len(line) - 2], n, nil    
}

// read integer
func (r *Resp) readInteger() (x int, y int, err error) {
    line, n, err := r.readLine()

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

func (r *Resp) Read() (Value, error) {
    _type, err := r.reader.ReadByte()

    if err != nil {
        fmt.Println("chere")
        return Value{}, err
    }

    switch _type {
        case ARRAY:
            return r.readArray()
        case BULK:
            return r.readBulk()
        default:
            fmt.Printf("Unknown type: %v", string(_type))
            return Value{}, nil
    }
}

func (r *Resp) readArray() (Value, error) {
    v := Value{}
    v.typ = "array"
    
    length, _, err := r.readInteger()

    if err != nil {
        return v, err
    }
    
    v.array = make([]Value, 0)

    for i := 0; i < length; i++ {
        val, err := r.Read()

        if err != nil {
            return v, err
        }

        v.array = append(v.array, val)
    }

    return v, nil
}

func (r *Resp) readBulk() (Value, error) {
    v := Value{}
    v.typ = "bulk"

    length, _, err := r.readInteger()

    if err != nil {
        return v, err
    }

    bulk := make([]byte, length)
    
    r.reader.Read(bulk) // bufio read
    v.bulk = string(bulk)

    // read trailing CRLF
    r.readLine()

    return v, nil
}

// Writer

type Writer struct {
    writer io.Writer
}

func NewWriter(w io.Writer) *Writer {
    return &Writer{writer: w}
}

func (w* Writer) Write(val Value) error {
    var bytes = val.Marshall()
    
    // calls io writer write
    _, err = w.writer.Write(bytes)

    if err != nil {
        return err
    }

    return nil
}

// Marshall

func (val Value) Marshall() []byte {
    switch val.typ {
        case "array":
            return val.marshallArray()
        case "bulk":
            return val.marshallBulk()
        case "string":
            return val.marshallString()
        case "null":
            return val.marshallNull()
        case "error":
            return val.marshallError()
        default:
            return []byte{}
    }
}

func (val Value) marshallString() []byte {
    var bytes []byte

    bytes = append(bytes, STRING)
    bytes = append(bytes, val.str...)
    bytes = append(bytes, '\r', '\n')

    return bytes
}

func (val Value) marshallBulk() []byte {
    var bytes []byte

    bytes = append(bytes, BULK)
    bytes = append(bytes, strconv.Itoa(len(val.bulk))...)
    bytes = append(bytes, '\r', '\n')
    bytes = append(bytes, val.bulk...)
    bytes = append(bytes, '\r', '\n')

    return bytes
}

func (val Value) marshallArray() []byte {
    var bytes []byte
    length := len(var.array)

    bytes = append(bytes, ARRAY)
    bytes = append(bytes, strconv.Itoa(length)...)
    bytes = append(bytes, '\r', '\n')

    for i := 0; i < length; i++ {
        bytes = append(bytes, val.array[i].Marshal()...)
    }

    return bytes
}

func (val Value) marshallError() []byte {
    var bytes []byte
   
    bytes = append(bytes, ERROR)
    bytes = append(bytes, val.str...)
    bytes = append(bytes, '\r', '\n')
    
    return bytes
}

func (val Value) marshallNull() []byte {
    var bytes []byte

    bytes = append(bytes, "$-1\r\n"...)

    return bytes
}
