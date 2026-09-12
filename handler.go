
var Handlers = map[string]func([]Value) Value {
    "PING": ping,
    "SET": set,
    "GET": get,
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
    val, ok = SETs[key]
    SETsMu.RUnlock()

    if !ok {
        return Value{typ: "error", str: "Error getting value from map, key does not exist"}
    }

    return Value{typ: "bulk", bulk: val}
}
