package main

import (
    "bufio"
    "fmt"
    "io"
    "strconv"
)

// reader

const (
    RESPstring = '+'
    RESPerror = '-'
    RESPinteger = ':'
    RESPbulk = '$'
    RESParray = '*'
)

// struct to hold all commands we recieve from the client
type Value struct {
    typ string
    str string
    num int
    bulk string
    array []value
}

// reader to help use read from the buffer and store in the value struct
type resp struct {
    reader *bufio.Reader
}
// initialize new resp object
func NewResp(rd io.Reader) *resp {
    return &resp{reader: bufio.NewReader(rd)}
}

// function to read byte until it reaches \r, return without \r\n
func (r *resp) readline() (line []byte, n int, err error) {
    for {
        b, err := r.reader.readbyte()

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
func (r *resp) readinteger() (x int, y int, err error) {
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

func (r *resp) read() (value, error) {
    _type, err := r.reader.readbyte()

    if err != nil {
        fmt.println("chere")
        return value{}, err
    }

    switch _type {
        case array:
            return r.readarray()
        case bulk:
            return r.readbulk()
        default:
            fmt.printf("unknown type: %v", string(_type))
            return value{}, nil
    }
}

func (r *resp) readarray() (value, error) {
    v := value{}
    v.typ = "array"
    
    length, _, err := r.readinteger()

    if err != nil {
        return v, err
    }
    
    v.array = make([]value, 0)

    for i := 0; i < length; i++ {
        val, err := r.read()

        if err != nil {
            return v, err
        }

        v.array = append(v.array, val)
    }

    return v, nil
}

func (r *resp) readbulk() (value, error) {
    v := value{}
    v.typ = "bulk"

    length, _, err := r.readinteger()

    if err != nil {
        return v, err
    }

    bulk := make([]byte, length)
    
	_, err := io.ReadFull(r.reader, bulk)
    v.bulk = string(bulk)

    // read trailing crlf
    r.readline()

    return v, nil
}

// writer

type writer struct {
    writer io.Writer
}

func NewWriter(w io.Writer) *writer {
    return &writer{writer: w}
}

func (w* writer) Write(val value) error {
    var bytes = val.marshall()
    
    // calls io writer write
    _, err = w.writer.Write(bytes)

    if err != nil {
        return err
    }

    return nil
}

// marshall

func (val value) marshall() []byte {
    switch val.typ {
        case "array":
            return val.marshallarray()
        case "bulk":
            return val.marshallbulk()
        case "string":
            return val.marshallstring()
        case "null":
            return val.marshallnull()
        case "error":
            return val.marshallerror()
        default:
            return []byte{}
    }
}

func (val value) marshallstring() []byte {
    var bytes []byte

    bytes = append(bytes, string)
    bytes = append(bytes, val.str...)
    bytes = append(bytes, '\r', '\n')

    return bytes
}

func (val value) marshallbulk() []byte {
    var bytes []byte

    bytes = append(bytes, bulk)
    bytes = append(bytes, strconv.Itoa(len(val.bulk))...)
    bytes = append(bytes, '\r', '\n')
    bytes = append(bytes, val.bulk...)
    bytes = append(bytes, '\r', '\n')

    return bytes
}

func (val value) marshallarray() []byte {
    var bytes []byte
    length := len(val.array)

    bytes = append(bytes, array)
    bytes = append(bytes, strconv.Itoa(length)...)
    bytes = append(bytes, '\r', '\n')

    for i := 0; i < length; i++ {
        bytes = append(bytes, val.array[i].marshal()...)
    }

    return bytes
}

func (val value) marshallerror() []byte {
    var bytes []byte
   
    bytes = append(bytes, error)
    bytes = append(bytes, val.str...)
    bytes = append(bytes, '\r', '\n')
    
    return bytes
}

func (val value) marshallnull() []byte {
    var bytes []byte

    bytes = append(bytes, "$-1\r\n"...)

    return bytes
}
