// this file contains functions for creating and converting javascript typed arrays and buffers.

package bridge

/*
#include "./include0_quickjs.h"

//---- TYPEDARRAY: SHARED BUFFER ----//

// forward declaration of the of the shared-buffer-freeing callback function, otherwise the compiler won't discover it.
extern void sharedArrayBufferFreeFunc(JSRuntime* rt, void* opaque_handle, void* first_byte_ptr);
*/
import "C"
import (
	fmt "fmt"
	runtime "runtime"
	unsafe "unsafe"
)

//----- TYPEDARRAY: TYPE CHECKS -----//

type TypedArrayEnum int // super set of `C.JSTypedArrayEnum`

const (
	TypedArrayInvalid   TypedArrayEnum = -1
	TypedArrayUint8C    TypedArrayEnum = C.JS_TYPED_ARRAY_UINT8C
	TypedArrayInt8      TypedArrayEnum = C.JS_TYPED_ARRAY_INT8
	TypedArrayUint8     TypedArrayEnum = C.JS_TYPED_ARRAY_UINT8
	TypedArrayInt16     TypedArrayEnum = C.JS_TYPED_ARRAY_INT16
	TypedArrayUint16    TypedArrayEnum = C.JS_TYPED_ARRAY_UINT16
	TypedArrayInt32     TypedArrayEnum = C.JS_TYPED_ARRAY_INT32
	TypedArrayUint32    TypedArrayEnum = C.JS_TYPED_ARRAY_UINT32
	TypedArrayBigInt64  TypedArrayEnum = C.JS_TYPED_ARRAY_BIG_INT64
	TypedArrayBigUint64 TypedArrayEnum = C.JS_TYPED_ARRAY_BIG_UINT64
	TypedArrayFloat16   TypedArrayEnum = C.JS_TYPED_ARRAY_FLOAT16
	TypedArrayFloat32   TypedArrayEnum = C.JS_TYPED_ARRAY_FLOAT32
	TypedArrayFloat64   TypedArrayEnum = C.JS_TYPED_ARRAY_FLOAT64
)

// test if your value is an instance of an `ArrayBuffer`.
func (arr *Value) IsArrayBuffer() bool {
	return arr.Instanceof(arr.ctx.valueCache.arrayBuffer)
}

// test if your value is an instance of a certain kind of `TypedArray`, specified by the `kind“ ([TypedArrayEnum]) enum option.
//
// make sure **not** to use [TypedArrayInvalid] for the `kind` parameter, otherwise the function will `panic`.
func (arr *Value) IsTypedArray(kind TypedArrayEnum) bool {
	cache := arr.ctx.valueCache
	switch kind {
	case TypedArrayUint8C:
		return arr.Instanceof(cache.uint8ClampedArray)
	case TypedArrayInt8:
		return arr.Instanceof(cache.int8Array)
	case TypedArrayUint8:
		return arr.Instanceof(cache.uint8Array)
	case TypedArrayInt16:
		return arr.Instanceof(cache.int16Array)
	case TypedArrayUint16:
		return arr.Instanceof(cache.uint16Array)
	case TypedArrayInt32:
		return arr.Instanceof(cache.int32Array)
	case TypedArrayUint32:
		return arr.Instanceof(cache.uint32Array)
	case TypedArrayBigInt64:
		return arr.Instanceof(cache.bigInt64Array)
	case TypedArrayBigUint64:
		return arr.Instanceof(cache.bigUint64Array)
	case TypedArrayFloat16:
		return arr.Instanceof(cache.float16Array)
	case TypedArrayFloat32:
		return arr.Instanceof(cache.float32Array)
	case TypedArrayFloat64:
		return arr.Instanceof(cache.float64Array)
	default:
		panic(fmt.Sprintf(`[Value.IsTypedArray]: received an invalid enum for the "kind" of typed array: "%d"`, kind))
	}
}

// identifies the _kind_ ([TypedArrayEnum]) of your javascript typed array object.
//
// if your object is _not_ an instance of a `TypedArray`, then `-1` ([TypedArrayInvalid]) will be returned.
func (arr *Value) IdentifyTypedArray() TypedArrayEnum {
	cache := arr.ctx.valueCache
	if arr.Instanceof(cache.uint8ClampedArray) {
		return TypedArrayUint8C
	}
	if arr.Instanceof(cache.int8Array) {
		return TypedArrayInt8
	}
	if arr.Instanceof(cache.uint8Array) {
		return TypedArrayUint8
	}
	if arr.Instanceof(cache.int16Array) {
		return TypedArrayInt16
	}
	if arr.Instanceof(cache.uint16Array) {
		return TypedArrayUint16
	}
	if arr.Instanceof(cache.int32Array) {
		return TypedArrayInt32
	}
	if arr.Instanceof(cache.uint32Array) {
		return TypedArrayUint32
	}
	if arr.Instanceof(cache.bigInt64Array) {
		return TypedArrayBigInt64
	}
	if arr.Instanceof(cache.bigUint64Array) {
		return TypedArrayBigUint64
	}
	if arr.Instanceof(cache.float16Array) {
		return TypedArrayFloat16
	}
	if arr.Instanceof(cache.float32Array) {
		return TypedArrayFloat32
	}
	if arr.Instanceof(cache.float64Array) {
		return TypedArrayFloat64
	}
	return TypedArrayInvalid
}

// represents a javascript `TypedArray`'s internal state, acquired via [Value.TypedArrayInfo].
type TypedArrayInfo struct {
	Buffer          *Value
	ByteOffset      uint
	ByteLength      uint
	BytesPerElement uint
}

// identify the internal state of a javascript `TypedArray`.
//
// make sure that your javascript value _is_ a typed array (either via [Value.IsTypedArray],
// or [Value.IdentifyTypedArray]), otherwise the function will panic.
//
// @should-free (the `Buffer` must be freed after being used, since it's a javascript object)
func (arr *Value) IdentifyTypedArrayInfo() TypedArrayInfo {
	var (
		byte_offset       C.size_t
		byte_length       C.size_t
		bytes_per_element C.size_t
	)
	buffer_ref := C.JS_GetTypedArrayBuffer(arr.ctx.ref, arr.ref, &byte_offset, &byte_length, &bytes_per_element)
	buffer := &Value{ctx: arr.ctx, ref: buffer_ref}
	if buffer.IsArrayBuffer() {
		return TypedArrayInfo{
			Buffer:          buffer,
			ByteOffset:      uint(byte_offset),
			ByteLength:      uint(byte_offset),
			BytesPerElement: uint(bytes_per_element),
		}
	}
	panic("[Value.IdentifyTypedArrayInfo]: received a non-TypedArray javascript value.")
}

//---- TYPEDARRAY: CONSTRUCTION  ----//

// create a new empty javascript typed array, with the given length (element count).
//
// @should-free
func (ctx *Context) NewTypedArray(kind TypedArrayEnum, length uint32) *Value {
	if kind < 0 {
		panic(fmt.Sprintf(`[Context.NewTypedArray]: received an invalid enum for the "kind" of typed array: "%d"`, kind))
	}
	js_kind := C.JSTypedArrayEnum(kind)
	js_length_int := ctx.NewUint32(length)
	// equivalent to the js-signature: `new TypedArray(length)`
	js_arr_ref := C.JS_NewTypedArray(ctx.ref, 1, &js_length_int.ref, js_kind)
	return &Value{ctx: ctx, ref: js_arr_ref}
}

// create a new javascript typed array, by copying over the `raw_data` bytes to it.
//
// @should-free
func (ctx *Context) NewTypedArrayFromBytes(kind TypedArrayEnum, raw_data []byte) *Value {
	js_buf := ctx.NewArrayBuffer(raw_data)
	defer js_buf.Free()
	return ctx.NewTypedArrayFromArrayBuffer(kind, js_buf)
}

// create a new javascript typed array, by embedding an underlying javascript `ArrayBuffer` to it.
//
// @should-free
func (ctx *Context) NewTypedArrayFromArrayBuffer(kind TypedArrayEnum, js_array_buffer *Value) *Value {
	if kind < 0 {
		panic(fmt.Sprintf(`[Context.NewTypedArrayFromArrayBuffer]: received an invalid enum for the "kind" of typed array: "%d"`, kind))
	}
	js_kind := C.JSTypedArrayEnum(kind)
	// equivalent to the js-signature: `new TypedArray(buffer)`
	js_arr_ref := C.JS_NewTypedArray(ctx.ref, 1, &js_array_buffer.ref, js_kind)
	return &Value{ctx: ctx, ref: js_arr_ref}
}

// create a new javascript `ArrayBuffer` by copying over the `raw_data` bytes to it.
//
// in general, _copying_ memory should be preferred over _shared_ memory (i.e. [Context.NewArrayBufferShared]),
// because then, the go-runtime will be free to do whatever it wants to do to the original `raw_data`,
// without the possibility of it being garbage collecting by go, and then having quickjs's c-code try to access a no-longer-valid memory spot.
// likewise, the quickjs runtime will be free to clear up this array buffer's memory, without causing problems with the go-runtime.
//
// @should-free
func (ctx *Context) NewArrayBuffer(raw_data []byte) *Value {
	var first_byte_ptr *byte = nil
	raw_data_len := len(raw_data)
	if raw_data_len > 0 {
		first_byte_ptr = &raw_data[0]
	}
	js_arr_ref := C.JS_NewArrayBufferCopy(ctx.ref, (*C.uint8_t)(first_byte_ptr), C.size_t(raw_data_len))
	return &Value{ctx: ctx, ref: js_arr_ref}
}

//export sharedArrayBufferFreeFunc
func sharedArrayBufferFreeFunc(rt *C.JSRuntime, opaque_pinner unsafe.Pointer, data_first_byte_ptr unsafe.Pointer) {
	if opaque_pinner != nil {
		pinner := (*runtime.Pinner)(opaque_pinner)
		pinner.Unpin() // unpinning the memory region, so that the go runtime can garbage collect it whenever.
	}
}

// create a new javascript `ArrayBuffer` that shares its memory with the provided `raw_data` slice.
//
// since only the original memory region of the `raw_data` is shared with quickjs,
// you should avoid performing actions that might result in the re-allocation of the underlying memory.
// this means that you should avoid expanding or shrinking the capacity/length of your `raw_data` slice.
//
// under the hood, we use `runtime.Pinner` to pin down the `&raw_data[0]` memory location,
// to prevent go's garbage collection from freeing up or moving this memory region.
// however, this does not mean that your `raw_data` _slice_ will necessarily still point to the `&raw_data[0]` memory if you perform length expansion/contractions.
// which is why you should avoid those operations if you want the _shared_ memory region to remain associated with your slice object.
//
// @should-free
func (ctx *Context) NewArrayBufferShared(raw_data []byte) *Value {
	raw_data_len := len(raw_data)
	if raw_data_len == 0 {
		// since `unsafe.Pointer(nil)` is not permitted, we will have to avoid this situation by branching into a separate case.
		// but as an alternative, we simply cheat by creating a non-shared buffer, since there is zero underlying shared data anyway.
		return ctx.NewArrayBuffer(raw_data)
	}
	first_byte_ptr := &raw_data[0]
	// below, we tell the go runtime not to garbage collect the actual memory region of `raw_data` until we unpin it.
	// this process is known as "pinning" the memory region. and freeing it up is known as "unpinning" the region.
	pinner := &runtime.Pinner{}
	pinner.Pin(first_byte_ptr)
	js_arr_ref := C.JS_NewArrayBuffer(
		ctx.ref, (*C.uint8_t)(first_byte_ptr), C.size_t(raw_data_len),
		(*C.JSFreeArrayBufferDataFunc)(C.sharedArrayBufferFreeFunc), unsafe.Pointer(pinner), (C.JS_BOOL)(1),
	)
	js_arr := &Value{ctx: ctx, ref: js_arr_ref}
	// memory free up trajectory: `js_arr.Free()` -> `C.JS_FreeValue(...)` -> `C.sharedArrayBufferFreeFunc(...)` -> `sharedArrayBufferFreeFunc(...)` -> done
	return js_arr
}

//----- TYPEDARRAY: CONVERSION  -----//

// converts both `ArrayBuffer`s and instances of `TypedArray`s into a slice of bytes, by copying their memory.
// this means that any modification performed on the returned slice will not be reflected on the javascript side.
//
// if you wish to obtain a slice pointing to a _shared_ memory region, consider using [Value.ToByteArrayShared].
// (make sure to read its instructions though, since you don't want dangling pointers happening either on the quickjs side, or the go side)
//
// TODO: show how to convert `[]byte` to numeric slices, such as `[]uint32`, `[]float64`, etc..., possibly without copying the existing slice's data.
// one challenge with multi-byte types is that we will have to keep quickjs's endianness and the go-runtime's endianness in mind (I think).
func (arr *Value) ToByteArray() []byte {
	shared_byte_slice := arr.ToByteArrayShared()
	go_managed_slice := make([]byte, len(shared_byte_slice))
	copy(go_managed_slice, shared_byte_slice)
	// no need to free `shared_byte_slice`, as it is managed/allocated by quickjs's c-code, and not us.
	return go_managed_slice
}

// converts both `ArrayBuffer`s and instances of `TypedArray`s into a slice of **shared** bytes (i.e. no memory copied).
// this means that any modification performed on the returned slice **will** be reflected on the javascript side.
//
// since the shared memory region is originally allocated by quickjs (with the exception of [Context.NewArrayBufferShared], but it still doesn't matter),
// the go runtime will not randomly claim that memory region and free it up. so, you're safe in that aspect.
// (i.e. there's no need for you to perform `(&runtime.Pinner{}).Pin()` on it)
//
// however, if quickjs decides to free up the memory (i.e. when js object associated with the memory has been freed),
// your byte slice's data pointer will become a dangling pointer, and accessing elements of the slice will possibly lead to memory corruption.
//
// thus, when using this method, make sure to keep close tabs on the associated js object that owns this memory region,
// so that you do not end up with the dangling pointer situation.
func (arr *Value) ToByteArrayShared() []byte {
	var typed_info TypedArrayInfo
	is_array_buffer := arr.IsArrayBuffer()
	if is_array_buffer {
		typed_info.Buffer = arr
		typed_info.ByteOffset = 0
		typed_info.ByteLength = 0
		typed_info.BytesPerElement = 1
	} else {
		typed_info = arr.IdentifyTypedArrayInfo() // will panic if `arr` is not a typed array.
		defer typed_info.Buffer.Free()
	}
	var buf_length C.size_t
	first_buf_byte_ptr := C.JS_GetArrayBuffer(typed_info.Buffer.ctx.ref, &buf_length, typed_info.Buffer.ref)
	// if the buffer's length is zero, then `first_byte_ptr` will likely be `nil`, so we will return an empty slice.
	if buf_length == 0 {
		return []byte{}
	}
	first_view_byte_ptr := unsafe.Add(unsafe.Pointer(first_buf_byte_ptr), typed_info.ByteOffset)
	view_byte_length := typed_info.ByteLength
	if is_array_buffer {
		view_byte_length = uint(buf_length)
	}
	shared_byte_slice := unsafe.Slice((*byte)(first_view_byte_ptr), C.int(view_byte_length))
	return shared_byte_slice
}

// - [ ] TODO: other miscellaneous classes to add support for: `Date`, `RegExp`, `JSON`, `Promise`.
// - [ ] TODO: classes to consider adding support for if they're not too difficult to implement: `GeneratorFunction`, `AsyncGeneratorFunction`, `Proxy`, `Reflect`.
// - [x] TODO: add the `IsArray`, `IsHashMap`, and `IsHashSet` functions.
// - [x] TODO: add the `NewArray`, `NewHashMap`, and `NewHashSet` functions.
// - [ ] TODO: add the `ToArray`, `ToHashMap`, and `ToHashSet` functions.

//------      TYPE CHECKS      ------//

func (val *Value) IsArray() bool   { return val != nil && C.JS_IsArray(val.ctx.ref, val.ref) == 1 }
func (val *Value) IsHashMap() bool { return val.Instanceof(val.ctx.valueCache.hashMap) }
func (val *Value) IsHashSet() bool { return val.Instanceof(val.ctx.valueCache.hashSet) }
func (val *Value) IsWeakMap() bool { return val.Instanceof(val.ctx.valueCache.weakMap) }
func (val *Value) IsWeakSet() bool { return val.Instanceof(val.ctx.valueCache.weakSet) }

//------     CONSTRUCTION      ------//

// - TODO: should I permit the user to pre-add entries?
//   since go lang does not permit optional args, we can't really have a single function that can do both: create an empty collection, or create one with existing items.
//   thus one possibility would be to write two versions of each collection class, one that creates an empty collection, like `NewEmptyArray()`,
//   and the other which creates a pre-filled collection, like `NewArray(items []*Value)`, or `NewArray(length uint)`.
//
// - TODO: also, what do we do with regards to accelerated method calls using [Atom]s?
//   for example, pushing 1000 objects into an array would be inefficient if we were to do `js_arr.CallMethod("push", item)` a thousand times.
//   it would be better if we did `js_arr.CallMethodAtom(push_str_atom, item)`,
//   but it would be even better if we were to obtain the `prototype.push` function, then call it via either `js_prototype_push.Call`,
//   or use the `C.JS_Call(...)` method if we already know the method's signature (i.e. no var-arg overhead).
//   finally, I think it would be better if we place the accelerated methods for builtin-js-objects in another package,
//   where it will have provide a go-struct for each collection type that will inherit the underlying [Value] via composition.
//   for example: `type ArrayValue struct { val *Value };` and the methods will be defined like: `func (arr *ArrayValue) push(item *Value) uint { return uint(C.JS_Call(...)) }`.
//   (why don't we permit multiple item pushes here? because that would defeat the point of the function, as it is meant to accelerate single pushes.
//   for multiple pushes, one can simply use the `Value.CallMethod` or the `Value.Call` methods with var args.)
//
// - TODO: [not relevant to this file]: bind the `JS_PrintValue` or `JSPrintValueWrite`, and the `JS_PrintValueSetDefaultOptions` functions,
//   and then, in your polyfill package, use it to polyfil `console.log` and make the output string get printed by `println()`
//
// - TODO: also implement a `Len` method that returns the length of `Array`s and `TypedArray`s, and the `size` of `Map`s and `Set`s, and the `byteLength` of `ArrayBuffer`s.

// create a new javascript `Array` object.
func (ctx *Context) NewArray() *Value {
	ref := C.JS_NewArray(ctx.ref)
	return &Value{ctx: ctx, ref: ref}
}

// create a new javascript `Map` object.
func (ctx *Context) NewHashMap() *Value {
	return ctx.valueCache.hashMap.CallConstructor()
}

// create a new javascript `Set` object.
func (ctx *Context) NewHashSet() *Value {
	return ctx.valueCache.hashSet.CallConstructor()
}

// create a new javascript `WeakMap` object.
func (ctx *Context) NewWeakMap() *Value {
	return ctx.valueCache.weakMap.CallConstructor()
}

// create a new javascript `WeakSet` object.
func (ctx *Context) NewWeakSet() *Value {
	return ctx.valueCache.weakSet.CallConstructor()
}

//------      CONVERSION       ------//

// TODO: I'm getting bored now
