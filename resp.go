package main

import (
    "bufio"
    "fmt"
    "io"
    "strconv"
)

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
    
    // _, err = io.ReadFull(r.reader, bulk)
    // 
    // if err != nil {
    //     return v, err
    // }

    r.reader.Read(bulk) // bufio read
    v.bulk = string(bulk)

    // read trailing CRLF
    r.readLine()

    return v, nil
}
