package luar

import (
	"fmt"
	"reflect"

	"github.com/yuin/gopher-lua"
)

var typeMetatable map[string]map[string]lua.LGFunction

func init() {
	typeMetatable = map[string]map[string]lua.LGFunction{
		"chan": {
			"__index":    chanIndex,
			"__tostring": chanToString,
			"__eq":       baseEqual,
		},
		"map": {
			"__index":    mapIndex,
			"__newindex": mapNewIndex,
			"__len":      mapLen,
			"__call":     mapCall,
			"__tostring": baseToString,
			"__eq":       baseEqual,
		},
		"ptr": {
			"__index":    ptrIndex,
			"__newindex": ptrNewIndex,
			"__tostring": ptrToString,
			"__eq":       baseEqual,
		},
		"slice": {
			"__index":    sliceIndex,
			"__newindex": sliceNewIndex,
			"__len":      sliceLen,
			"__tostring": baseToString,
			"__eq":       baseEqual,
		},
		"struct": {
			"__index":    structIndex,
			"__newindex": structNewIndex,
			"__tostring": baseToString,
		},
		"type": {
			"__call":     typeCall,
			"__tostring": typeToString,
		},
	}
}

func baseToString(L *lua.LState) int {
	ud := L.CheckUserData(1)
	value := reflect.ValueOf(ud.Value)

	str := fmt.Sprintf("userdata: luar: %s %+v (%p)", value.Type(), value.Interface(), ud.Value)
	L.Push(lua.LString(str))
	return 1
}

func baseEqual(L *lua.LState) int {
	ud1 := L.CheckUserData(1)
	ud2 := L.CheckUserData(2)
	L.Push(lua.LBool(ud1.Value == ud2.Value))
	return 1
}

func ensureMetatable(L *lua.LState) *lua.LTable {
	const metatableKey = lua.LString("github.com/layeh/gopher-luar")
	v := L.G.Registry.RawGetH(metatableKey)
	if v != lua.LNil {
		return v.(*lua.LTable)
	}
	newTable := L.NewTable()
	for typeName, typeMethods := range typeMetatable {
		typeTable := L.NewTable()
		typeTable.RawSetH(lua.LString("__metatable"), lua.LTrue)
		for methodName, methodFunc := range typeMethods {
			typeTable.RawSetH(lua.LString(methodName), L.NewFunction(methodFunc))
		}
		newTable.RawSetH(lua.LString(typeName), typeTable)
	}
	L.G.Registry.RawSetH(metatableKey, newTable)
	return newTable
}

// New creates and returns a new lua.LValue for the given value.
//
// The following types are supported:
//  reflect.Kind    gopher-lua Type
//  nil             LNil
//  Bool            LBool
//  Int             LNumber
//  Int8            LNumber
//  Int16           LNumber
//  Int32           LNumber
//  Int64           LNumber
//  Uint            LNumber
//  Uint8           LNumber
//  Uint32          LNumber
//  Uint64          LNumber
//  Float32         LNumber
//  Float64         LNumber
//  Chan            *LUserData
//  Interface       *LUserData
//  Func            *lua.LFunction
//  Map             *LUserData
//  Ptr             *LUserData
//  Slice           *LUserData
//  String          LString
//  Struct          *LUserData
//  UnsafePointer   *LUserData
func New(L *lua.LState, value interface{}) lua.LValue {
	if value == nil {
		return lua.LNil
	}
	if lval, ok := value.(lua.LValue); ok {
		return lval
	}
	table := ensureMetatable(L)

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Bool:
		return lua.LBool(val.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return lua.LNumber(float64(val.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return lua.LNumber(float64(val.Uint()))
	case reflect.Float32, reflect.Float64:
		return lua.LNumber(val.Float())
	case reflect.Chan:
		ud := L.NewUserData()
		ud.Value = val.Interface()
		ud.Metatable = table.RawGetH(lua.LString("chan"))
		return ud
	case reflect.Func:
		return funcWrapper(L, val)
	case reflect.Interface:
		ud := L.NewUserData()
		ud.Value = val.Interface()
		return ud
	case reflect.Map:
		ud := L.NewUserData()
		ud.Value = val.Interface()
		ud.Metatable = table.RawGetH(lua.LString("map"))
		return ud
	case reflect.Ptr:
		ud := L.NewUserData()
		ud.Value = val.Interface()
		ud.Metatable = table.RawGetH(lua.LString("ptr"))
		return ud
	case reflect.Slice:
		ud := L.NewUserData()
		ud.Value = val.Interface()
		ud.Metatable = table.RawGetH(lua.LString("slice"))
		return ud
	case reflect.String:
		return lua.LString(val.String())
	case reflect.Struct:
		ud := L.NewUserData()
		ud.Value = val.Interface()
		ud.Metatable = table.RawGetH(lua.LString("struct"))
		return ud
	case reflect.UnsafePointer:
		ud := L.NewUserData()
		ud.Value = val.Interface()
		return ud
	}
	return nil
}

// NewType returns a new type creator for the given value's type.
//
// When the lua.LValue is called, a new value will be created that is the
// same type as value's type.
func NewType(L *lua.LState, value interface{}) lua.LValue {
	table := ensureMetatable(L)

	valueType := reflect.TypeOf(value)
	ud := L.NewUserData()
	ud.Value = valueType
	ud.Metatable = table.RawGetH(lua.LString("type"))
	return ud
}

func lValueToReflect(v lua.LValue, hint reflect.Type) reflect.Value {
	switch converted := v.(type) {
	case lua.LBool:
		return reflect.ValueOf(bool(converted))
	case lua.LChannel:
		return reflect.ValueOf(converted)
	case lua.LNumber:
		return reflect.ValueOf(converted).Convert(hint)
	case *lua.LFunction:
		return reflect.ValueOf(converted)
	case *lua.LNilType:
		return reflect.Zero(hint)
	case *lua.LState:
		return reflect.ValueOf(converted)
	case lua.LString:
		return reflect.ValueOf(string(converted))
	case *lua.LTable:
		return reflect.ValueOf(converted)
	case *lua.LUserData:
		return reflect.ValueOf(converted.Value)
	}
	panic("fatal lValueToReflect error")
	return reflect.Value{}
}
