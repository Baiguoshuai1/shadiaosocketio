package shadiaosocketio

import (
	"encoding/json"
	"errors"
	"reflect"
)

type caller struct {
	Func   reflect.Value
	NumInt int
	NumOut int
}

var (
	ErrorCallerNotFunc     = errors.New("f is not function")
	ErrorCallerNot2Args    = errors.New("f maximum number of args is 5")
	ErrorCallerMaxOneValue = errors.New("f number of args returned should be 1")
)

/**
Parses function passed by using reflection, and stores its representation
for further call on message or ack
*/
func newCaller(f interface{}) (*caller, error) {
	fVal := reflect.ValueOf(f)
	if fVal.Kind() != reflect.Func {
		return nil, ErrorCallerNotFunc
	}

	fType := fVal.Type()
	if fType.NumOut() > 1 {
		panic(ErrorCallerMaxOneValue)
	}
	if fType.NumIn() > 5 {
		panic(ErrorCallerNot2Args)
	}

	curCaller := &caller{
		Func:   fVal,
		NumInt: fType.NumIn(),
		NumOut: fType.NumOut(),
	}

	return curCaller, nil
}

func (c *caller) getArgType(index int) interface{} {
	return reflect.New(c.Func.Type().In(index)).Interface()
}

func (c *caller) getOutType(index int) interface{} {
	return reflect.New(c.Func.Type().Out(index)).Interface()
}

func (c *caller) callFunc(h *Channel, args ...interface{}) []reflect.Value {
	arr := make([]reflect.Value, 0, 1+c.NumInt)
	arr = append(arr, reflect.ValueOf(h))

	for i := 0; i < c.NumInt-1; i++ { // * 1 2   // x{0} y{1}
		data := c.getArgType(i + 1)

		if i > len(args)-1 {
			arr = append(arr, reflect.ValueOf(data).Elem())
			continue
		}

		marshal, _ := json.Marshal(args[i])
		err := json.Unmarshal(marshal, &data)
		if err != nil {
			panic(err)
		}

		arr = append(arr, reflect.ValueOf(data).Elem())
	}

	return c.Func.Call(arr)
}
