// this file contains a wrapper for javascript `Object`s.

package bridge

/*
#include "./include0_quickjs.h"

// TODO ISSUE: for some reason `JS_GetPropertyInt64` is not exported in the header file, even though it exists in the c-code.
//   hence, we will declare its signature over here.
// TODO: actually, this won't work with the static library compilation mode since it does not export the `JS_GetPropertyInt64` function.
// JSValue JS_GetPropertyInt64(JSContext *ctx, JSValueConst this_obj, int64_t idx);
*/
import "C"
import (
	fmt "fmt"
	unsafe "unsafe"
)

// set a javascript `Object`'s property `prop` to a certain value `val`.
//
// since the host object will take ownership of the `val` (i.e. it will free it automatically upon the host's destruction),
// you should ensure that you do not free it yourself; unless it is intended to outlive its host via the [Value.Dupe] method.
//
// more precisely, the [Value.Set] method does not increment the reference counting of the provided `val`,
// however, it will decrement its reference count once the host object is destroyed (i.e. its reference count drops to `0`),
// or if the property gets replaced by a new property.
//
// @ownership-transfer
func (obj *Value) Set(prop string, val *Value) {
	cstr_ptr := C.CString(prop)
	defer C.free(unsafe.Pointer(cstr_ptr))
	success := C.JS_SetPropertyStr(obj.ctx.ref, obj.ref, cstr_ptr, val.ref)
	// success is either `-1` (exception), `0` (false), or `1` (true).
	if success < 0 {
		panic(fmt.Sprintf(`[Object.Set]: setting the value of the property "%s" resulted in an exception.`, prop))
	}
}

// get the value of a javascript `Object`'s property `prop`.
//
// quickjs increments the reference count of the returned [Value] whenever it is acquired this way.
// in other words, you must call the [Value.Free] method once you have used the returned value
// (supposing that you have not transferred its owenership to a _different_ object via the [Value.Set] method).
//
// @should-free
func (obj *Value) Get(prop string) *Value {
	cstr_ptr := C.CString(prop)
	defer C.free(unsafe.Pointer(cstr_ptr))
	return &Value{ctx: obj.ctx, ref: C.JS_GetPropertyStr(obj.ctx.ref, obj.ref, cstr_ptr)}
}

// dictates whether or not an object has a certain value property `prop`.
func (obj *Value) Has(prop string) bool {
	prop_atom := obj.ctx.NewAtom(prop)
	defer prop_atom.Free()
	return obj.HasAtom(prop_atom)
}

// delete/remove a javascript `Object`'s property `prop`, and have it freed (the property field will become _uninitialized_ afterwards).
//
// a `true` returned value indicates that the `prop` had been initialized prior to being deleted,
// while a `false` would indicate that the `prop` had never been initialized before.
func (obj *Value) Delete(prop string) bool {
	prop_atom := obj.ctx.NewAtom(prop)
	defer prop_atom.Free()
	return obj.DeleteAtom(prop_atom)
}

// set a javascript `Object`'s index property `idx` to a certain value `val`.
//
// since the host object will take ownership of the `val` (i.e. it will free it automatically upon the host's destruction),
// you should ensure that you do not free it yourself; unless it is intended to outlive its host via the [Value.Dupe] method.
//
// more precisely, the [Value.SetIdx] method does not increment the reference counting of the provided `val`,
// however, it will decrement its reference count once the host object is destroyed (i.e. its reference count drops to `0`).
//
// @ownership-transfer
func (obj *Value) SetIdx(idx int64, val *Value) {
	success := C.JS_SetPropertyInt64(obj.ctx.ref, obj.ref, C.int64_t(idx), val.ref)
	// success is either `-1` (exception), `0` (false), or `1` (true).
	if success < 0 {
		panic(fmt.Sprintf(`[Object.Set]: setting the value of the numeric index "%d" resulted in an exception.`, idx))
	}
}

const max_int32 int64 = 0x7FFFFFFF

// get a javascript `Object`'s property at index `idx`.
//
// quickjs increments the reference count of the returned [Value] whenever it is acquired this way.
// in other words, you must call the [Value.Free] method once you have used the returned value
// (supposing that you have not transferred its owenership to a _different_ object via the [Value.Set] method).
//
// TODO: only positive and 32-bit indexes currently work due to quickjs not exporting the relevant functions.
// anything outside this range will cause a fatal panic.
//
// @should-free
func (obj *Value) GetIdx(idx int64) *Value {
	// here, we recreate the inner logic of `JS_GetPropertyInt64` since it is not an exported function.
	if (idx >= 0) && (idx <= max_int32) {
		ctx := obj.ctx
		return &Value{ctx: ctx, ref: C.JS_GetPropertyUint32(obj.ctx.ref, obj.ref, C.uint32_t(idx))}
	}
	panic("[Value.GetIdx]: TODO ISSUE: negative indexes and those greater than `uint32` have not been implemented due to the inavailability of `JS_NewAtomInt64` and/or `JS_GetPropertyInt64` in the header file.")
	// atom_prop := C.JSAtom{}
	// defer C.JS_FreeAtom(ctx, atom_prop)
	// val_ref := C.JS_GetProperty(ctx.ref, obj.ref, atom_prop)
	// return &Value{ctx: ctx, ref: val_ref}
}

// dictates whether or not an object has a property at a certain index `idx`.
func (obj *Value) HasIdx(idx int64) bool {
	if (idx >= 0) && (idx <= max_int32) {
		prop_atom := obj.ctx.NewAtomIdx(uint32(idx))
		defer prop_atom.Free()
		return obj.HasAtom(prop_atom)
	}
	panic("[Value.HasIdx]: TODO ISSUE: negative indexes and those greater than `uint32` have not been implemented due to the inavailability of `JS_NewAtomInt64` and/or `JS_GetPropertyInt64` in the header file.")
}

// delete/remove a javascript `Object`'s property at index `idx`, and have it freed (the property field will become _uninitialized_ afterwards).
//
// a `true` returned value indicates that the `idx` index had been initialized prior to being deleted,
// while a `false` would indicate that the `idx` index had never been initialized before.
func (obj *Value) DeleteIdx(idx int64) bool {
	if (idx >= 0) && (idx <= max_int32) {
		prop_atom := obj.ctx.NewAtomIdx(uint32(idx))
		defer prop_atom.Free()
		return obj.DeleteAtom(prop_atom)
	}
	panic("[Value.DeleteIdx]: TODO ISSUE: negative indexes and those greater than `uint32` have not been implemented due to the inavailability of `JS_NewAtomInt64` and/or `JS_GetPropertyInt64` in the header file.")
}

// call a javascript `Object`'s method `method_name`, with the given arguments `args`.
func (obj *Value) CallMethod(method_name string, args ...*Value) *Value {
	js_fn := obj.Get(method_name)
	defer js_fn.Free()
	return js_fn.Call(obj, args...)
}
