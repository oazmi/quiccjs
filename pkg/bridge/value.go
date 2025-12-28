// this file contains the wrapper for quickjs's `Value` struct,
// in addition to `js<--->go` conversion functions for most javascript primitives.
//
// below is a table of primitives that are covered in this file (rip if you're reading this in vscode):
//
// | js-type         | go-type(s)                            |
// |-----------------|---------------------------------------|
// | `number`        | `int32`, `uint32`, `int64`, `float64` |
// | `bigint`        | `int64`, `uint64`, `math/big.Int`     |
// | `string`        | `string`                              |
// | `boolean`       | `bool`                                |
// | `null`          | _N/A_ (cannot be represented)         |
// | `undefined`     | _N/A_ (cannot be represented)         |
// | _uninitialized_ | _N/A_ (cannot be represented)         |
//
// > [!note]
// > all of these js-primitives do not need to have their [Value] [Value.Free]d after creation, aside from the `string` type.
//
// TODO: implement a wrapper for js-symbols.

package bridge

/*
#include "./include0_quickjs.h"
*/
import "C"
import (
	math_big "math/big"
	unsafe "unsafe"
)

// this represents a quickjs javascript value.
//
//   - since quickjs uses reference counting for garbage collection, make sure to always call the [Free]
//     method when a javascript object is going out of scope to decrement its reference counting.
//   - do note that only "complex" c-objects need to be freed.
//     `number`, `null`, `undefined`, and `boolean` do not need to be freed.
//     but `string`s, `Object`s, `Array`s, `Promise`s, `Function`s, etc... all need to be freed up.
//   - moreover, when you create/duplicate a new reference to the same jsvalue,
//     you should use the [Dupe] method to increment to increment the reference counting.
type Value struct {
	// reference to the `Context` hosting this value.
	ctx *Context
	// reference (or rather, composition) of the underlying c-based quickjs value.
	ref C.JSValue
}

// decrements the reference count of a javascript object.
//
// once its reference count drops to `0`, the object's memory will be cleared and its child objects
// (i.e. js-properties) will have their reference counting decremented by one as well.
//
// to increment the reference count, use the [Value.Dupe] method.
func (val *Value) Free() {
	// a memory freeing operation can only apply when a context and value is present.
	// when it isn't, there's, nothing to free.
	if val == nil || val.ctx == nil {
		return
	}
	C.JS_FreeValue(val.ctx.ref, val.ref)
}

// decrements the reference count of a javascript object _when_ its [Context] exits/frees up (i.e. when [Context.Free] is called).
//
// this is opposed to freeing up the value _immediately_ via [Value.Free].
// it is intended for long lived objects, such as polyfills (like `fetch`, `TextEncoder`, etc...),
// that should be freed upon the context's destruction.
func (val *Value) FreeOnExit() {
	val.ctx.freeUpList = append(val.ctx.freeUpList, val)
}

// increments the reference count of a javascript object and duplicates its [Value] wrapper.
//
// this operation should be performed _not_ when an ownership transfer is happening (such as via the [Value.Set] method),
// but rather be performed when the ownership of a js-value is being _duplicated_ (i.e. being co-owned by two or more scopes/objects).
//
// it is quite rare for this method to be used outside of the internal library logic,
// so be sure that your logic is sound if you're using this method.
//
// to decrement the reference count, use the [Value.Free] method.
func (val *Value) Dupe() *Value {
	// the operation only takes place on non-nil values and contexts.
	if val == nil || val.ctx == nil {
		return nil
	}
	return &Value{ctx: val.ctx, ref: C.JS_DupValue(val.ctx.ref, val.ref)}
}

// get the [Context] of the value.
func (val *Value) GetContext() *Context {
	return val.ctx
}

//------      TYPE CHECKS      ------//

func (val *Value) IsNumber() bool        { return val != nil && C.JS_IsNumber(val.ref) == 1 }
func (val *Value) IsBigInt() bool        { return val != nil && C.JS_IsBigInt(val.ctx.ref, val.ref) == 1 }
func (val *Value) IsBool() bool          { return val != nil && C.JS_IsBool(val.ref) == 1 }
func (val *Value) IsNull() bool          { return val != nil && C.JS_IsNull(val.ref) == 1 }
func (val *Value) IsUndefined() bool     { return val != nil && C.JS_IsUndefined(val.ref) == 1 }
func (val *Value) IsException() bool     { return val != nil && C.JS_IsException(val.ref) == 1 }
func (val *Value) IsUninitialized() bool { return val != nil && C.JS_IsUninitialized(val.ref) == 1 }
func (val *Value) IsString() bool        { return val != nil && C.JS_IsString(val.ref) == 1 }
func (val *Value) IsSymbol() bool        { return val != nil && C.JS_IsSymbol(val.ref) == 1 }
func (val *Value) IsObject() bool        { return val != nil && C.JS_IsObject(val.ref) == 1 }
func (val *Value) IsError() bool         { return val != nil && C.JS_IsError(val.ctx.ref, val.ref) == 1 }
func (val *Value) IsFunction() bool      { return val != nil && C.JS_IsFunction(val.ctx.ref, val.ref) == 1 }
func (val *Value) IsConstructor() bool {
	// bloody gofmt won't let me place it in a single line.
	return val != nil && C.JS_IsConstructor(val.ctx.ref, val.ref) == 1
}

//------     MISCELLANEOUS     ------//

// get the `globalThis` javascript object.
//
// > [!important]
// > do **NOT** free the returned `globalThis` object, as it is internally cached,
// > and always has its reference count set to `1` throughout the execution of your program.
func (ctx *Context) GetGlobalThis() *Value {
	return ctx.valueCache.globalThis
}

//------        STRINGS        ------//

// create a new javascript string from a go-string.
//
// do note that the string can contain null characters (i.e. `"\x00"`) without issues.
//
// @should-free
func (ctx *Context) NewString(str string) *Value {
	cstr_len := C.size_t(len(str))
	// copy and allocate a new string on the c-heap.
	cstr_ptr := C.CString(str)
	// free the pointer to the string in the c-heap once it has been used.
	// note that `unsafe.Pointer` needs to be used here, despite `cstr_ptr` already being a pointer, because `C.free` accepts a generic `*void`,
	// but our c-pointer is a `*char`, and go does not permit casting of `*char` to `*void` unless explicitly done via the `unsafe.Pointer` function.
	defer C.free(unsafe.Pointer(cstr_ptr))
	return &Value{ctx: ctx, ref: C.JS_NewStringLen(ctx.ref, cstr_ptr, cstr_len)}
}

// returns the `string` representation of a value.
//
// note that the returned string may contain null characters (i.e. `"\x00"`) if its javascript counterpart had null characters as well.
func (val *Value) ToString() string {
	var cstr_len C.size_t
	// the `JS_ToCStringLen` function returns a c-heap pointer to the string, in addition to also writing the byte-length of the string into the `cstr_len` variable.
	cstr_ptr := C.JS_ToCStringLen(val.ctx.ref, &cstr_len, val.ref)
	// since the string that was created was by quickjs's allocator, it should be freed via its allocator as well, instead of `C.free` from the `<stdlib.h>`.
	defer C.JS_FreeCString(val.ctx.ref, cstr_ptr)
	// the reason for using `GoStringN` instead of `GoString` is that the returned string may contain null character, which we would want to include.
	// however, `GoString` stops reading at the first null character as that's what terminates a proper c-string.
	return C.GoStringN(cstr_ptr, C.int(cstr_len))
}

//------       NULLABLES       ------//

// create a new javascript `null` value.
//
// note that it does not need to be freed afterwards.
func (ctx *Context) NewNull() *Value {
	return &Value{ctx: ctx, ref: C.JS_NULL}
}

// create a new javascript `undefined` value.
//
// note that it does not need to be freed afterwards.
func (ctx *Context) NewUndefined() *Value {
	return &Value{ctx: ctx, ref: C.JS_UNDEFINED}
}

// create a new "uninitialized" javascript value.
// it can be used for setting certain fields to "uninitiallized" (i.e. equivalent to the javascript `delete` operator).
//
// note that it does not need to be freed afterwards.
func (ctx *Context) NewUninitialized() *Value {
	return &Value{ctx: ctx, ref: C.JS_UNINITIALIZED}
}

//------       BOOLEANS        ------//

// create a new javascript `boolean` value.
//
// note that it does not need to be freed afterwards.
func (ctx *Context) NewBool(state bool) *Value {
	if state {
		return &Value{ctx: ctx, ref: C.JS_TRUE}
	}
	return &Value{ctx: ctx, ref: C.JS_FALSE}
}

// returns the truthiness of a javascript value.
//
// note that it does not need to be freed afterwards.
func (val *Value) ToBool() bool {
	// TODO: `JS_ToBool` may return `-1` when an exception is encountered.
	// right now, I'm not handling that case, but what if I wanted to in the future? should I just call `panic()` and wipe my hands?
	return C.JS_ToBool(val.ctx.ref, val.ref) == 1
}

//------        NUMBERS        ------//

// create a new javascript `number` value.
//
// note that it does not need to be freed afterwards.
func (ctx *Context) NewInt32(value int32) *Value {
	return &Value{ctx: ctx, ref: C.JS_NewInt32(ctx.ref, C.int32_t(value))}
}

// create a new javascript `number` value.
//
// note that it does not need to be freed afterwards.
func (ctx *Context) NewUint32(value uint32) *Value {
	return &Value{ctx: ctx, ref: C.JS_NewUint32(ctx.ref, C.uint32_t(value))}
}

// create a new javascript `number` value.
//
// note that it does not need to be freed afterwards.
func (ctx *Context) NewInt64(value int64) *Value {
	return &Value{ctx: ctx, ref: C.JS_NewInt64(ctx.ref, C.int64_t(value))}
}

// create a new javascript `number` value.
//
// note that it does not need to be freed afterwards.
// TODO: `JS_NewUint64` does not exist, but should I create a `JS_NewFloat64` wrapper over it?
//   similar to how quickjs uses `JS_NewFloat64` internally for `JS_NewInt64`? (i.e. thereby not truly being `int64`. after all, js max integer is 53bits long)
// func (ctx *Context) NewUint64(value uint64) *Value {
// 	return &Value{ctx: ctx, ref: C.JS_NewUint64(ctx.ref, C.uint64_t(value))}
// }

// create a new javascript `bigint` value.
//
// @should-free
func (ctx *Context) NewBigInt64(value int64) *Value {
	return &Value{ctx: ctx, ref: C.JS_NewBigInt64(ctx.ref, C.int64_t(value))}
}

// create a new javascript `bigint` value.
//
// @should-free
func (ctx *Context) NewBigUint64(value uint64) *Value {
	return &Value{ctx: ctx, ref: C.JS_NewBigUint64(ctx.ref, C.uint64_t(value))}
}

// create a new javascript `bigint` value from go's [math_big.Int].
//
// @should-free
func (ctx *Context) NewBigInt(value *math_big.Int) *Value {
	js_val, err := ctx.Eval(value.Text(10) + "n")
	if err != nil {
		panic(err)
	}
	return js_val
}

// create a new javascript `number` value.
//
// note that it does not need to be freed afterwards.
func (ctx *Context) NewFloat64(value float64) *Value {
	return &Value{ctx: ctx, ref: C.JS_NewFloat64(ctx.ref, C.double(value))}
}

// returns the `int32` value of the value.
func (val *Value) ToInt32() int32 {
	cval := C.int32_t(0)
	C.JS_ToInt32(val.ctx.ref, &cval, val.ref)
	return int32(cval)
}

// returns the `uint32` value of the value.
func (val *Value) ToUint32() uint32 {
	cval := C.uint32_t(0)
	C.JS_ToUint32(val.ctx.ref, &cval, val.ref)
	return uint32(cval)
}

// returns the `int64` value of the value.
func (val *Value) ToInt64() int64 {
	cval := C.int64_t(0)
	C.JS_ToInt64(val.ctx.ref, &cval, val.ref)
	return int64(cval)
}

// returns the `int64` value of a `bigint`.
func (val *Value) ToBigInt64() int64 {
	cval := C.int64_t(0)
	C.JS_ToBigInt64(val.ctx.ref, &cval, val.ref)
	return int64(cval)
}

// returns a [math_big.Int] representation of a javascript `bigint`. a `nil` is returned in case it fails.
func (val *Value) ToBigInt() *math_big.Int {
	if !val.IsBigInt() {
		return nil
	}
	bigint, _ := new(math_big.Int).SetString(val.ToString(), 10) // the trailing "n" is not included in the string.
	return bigint
}

// returns the `float64` value of the value.
func (val *Value) ToFloat64() float64 {
	cval := C.double(0)
	C.JS_ToFloat64(val.ctx.ref, &cval, val.ref)
	return float64(cval)
}

//------        SYMBOLS        ------//

// create a new javascript `Symbol` with an optional javascript string description.
// if you don't want any description, simply enter `nil` for it.
//
// @should-free
func (ctx *Context) NewSymbol(js_string_description *Value) *Value {
	if js_string_description == nil {
		return ctx.valueCache.symbol.Call(nil)
	}
	return ctx.valueCache.symbol.Call(nil, js_string_description)
}
