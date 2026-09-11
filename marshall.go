package main

import (
    "fmt"
)

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
