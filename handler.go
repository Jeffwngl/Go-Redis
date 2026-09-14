package main

import (
	"sync"
)

var Handlers = map[string]func([]Value) Value {
    "PING": ping,
    "SET": set,
    "GET": get,
    "HSET": hset,
    "HGET": hget,
    "HGETALL": hgetall,
}

func ping(args []Value) Value {
    return Value{typ: "string", str: "PONG"}
}

// sets map
var SETs = map[string]string{}
// use RWMutex so that server can handle requests concurrently
// so that SETs map is not modified by multiple threads at the same time
var SETsMu = sync.RWMutex{}

// handles: SET key value
/*
args := []Value{
    {typ: "bulk", bulk: "key"},
    {typ: "bulk", bulk: "value"},
}
*/
func set(args []Value) Value {
    if len(args) != 2 {
        return Value{typ: "error", str: "Wrong number of arguments for 'set' command, expected 2"}
    }

    key := args[0].bulk
    val := args[1].bulk

    SETsMu.Lock()
    SETs[key] = val
    SETsMu.Unlock()

    return Value{typ: "string", str: "OK"}
}

// handles: GET key
func get(args []Value) Value {
    if len(args) != 1 {
        return Value{typ: "error", str: "Wrong number of arguments for 'get' command, expected 1"}
    }

    key := args[0].bulk

    SETsMu.RLock()
    val, ok := SETs[key]
    SETsMu.RUnlock()

    if !ok {
        return Value{typ: "error", str: "Error getting value from map, key does not exist"}
    }

    return Value{typ: "bulk", bulk: val}
}


var HSETs = map[string]map[string]string{}
var HSETsMu = sync.RWMutex{}

func hset(args []Value) Value {
    if len(args) != 3 {
        return Value{typ: "error", str: "Wrong number of arguments for 'hset' command, expected 3"}
    }

    hash := args[0].bulk
    key := args[1].bulk
    val := args[2].bulk

    HSETsMu.Lock()

    _, ok := HSETs[hash]

    if !ok {
        HSETs[hash] = map[string]string{}
    }

    HSETs[hash][key] = val

    HSETsMu.Unlock()

    return Value{typ: "string", str: "OK"}
}

func hget(args []Value) Value {
    if len(args) != 2 {
        return Value{typ: "error", str: "Wrong number of arguments for 'hget' command, expected 2"}
    }

    hash := args[0].bulk
    key := args[1].bulk

    HSETsMu.RLock()
    val, ok = HSETs[hash][key]
    HSETsMu.RUnlock()
    
    if !ok {
        return Value{typ: "error", str: "Error getting value from map"}
    }

    return Value{typ: "bulk", bulk: val}
}

func hgetall(args []Value) Value {
    if len(args) != 1 {
        return Value{typ: "error", str: "Wrong number of arguments for 'hgetall' command, expected 1"}
    }

    res := []Value{}

    hash := args[0].bulk

	for key, val := HSETs[hash] {
		res = append(
			res,
			Value{typ: "bulk", bulk: key},
			Value{typ: "bulk", bulk: val}
		)
	}
	
	return Value{typ: "array", array: res}
}

