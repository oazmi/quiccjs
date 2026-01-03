// this file contains a wrapper for javascript `Object`s.
//
// TODO: implement `Value.GetSymbol`, `Value.SetSymbol`, `Value.HasSymbol`, and `Value.DeleteSymbol` methods.
// TODO: in fact, consider creating a new struct for symbols that encapsulates their atomic value.
// doing so would mean that we won't have to perform back and forth conversions between `*Value` and `*Atom` when trying to use a symbol as a property key.
// it would also simplify the implementations of `Value.GetSymbol`, `Value.HasSymbol`, etc..., and the internal `*Atom` will not need to be constantly cleared up every time.
// important: inside the `Symbol` struct, consider storing `ref: C.JSAtom`, rather than storing `atom: *Atom` for a more compact struct.
// TODO: consider adding "As" methods, like `Value.AsAtom`, or `Atom.AsString`, etc...
// these methods will perform the same action as their "To" counterparts, but also consume/take-ownership of the original struct
// (whether it's through embedding a portion of the original struct into another struct (such as the `Atom.ref` field), or by freeing up the original struct after the conversion).

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

// create a new empty javascript `Object`.
//
// @should-free
func (ctx *Context) NewObject() *Value {
	return &Value{ctx: ctx, ref: C.JS_NewObject(ctx.ref)}
}

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

// TODO implement typeof, either here or inside `./value.go`.

// check if an object `obj` is an instance of a class constructor `cls`.
func (obj *Value) IsInstanceOf(cls *Value) bool {
	if obj == nil || cls == nil || cls.IsUndefined() {
		return false
	}
	success := C.JS_IsInstanceOf(obj.ctx.ref, obj.ref, cls.ref)
	// success is either `-1` (exception), `0` (false), or `1` (true).
	if success >= 0 {
		return success == 1
	}
	panic(`[Value.IsInstanceOf]: checking for "instanceof" resulted in an exception.`)
}

// get the prototype object of a javascript object (analogous to the `Object.getPrototypeOf(obj)` static function).
func (obj *Value) GetPrototypeOf() *Value {
	ref := C.JS_GetPrototype(obj.ctx.ref, obj.ref)
	return &Value{ctx: obj.ctx, ref: ref}
}

// set the prototype of a javascript object. (analogous to the `Object.setPrototypeOf(obj, proto)` static function)
//
// the returned value indicates whether or not the operation was successful
// (you may encounter a `false` when setting the prototype of a forbidden object, such as `Object.prototype` itself).
//
// note that we don't return back the original `obj` due to the risk that the user might double-free the returned value.
func (obj *Value) SetPrototypeTo(js_proto *Value) bool {
	success := C.JS_SetPrototype(obj.ctx.ref, obj.ref, js_proto.ref)
	// success is either `-1` (exception), `0` (false), or `1` (true).
	if success >= 0 {
		return success == 1
	}
	panic(`[Value.SetPrototypeTo]: encountered an exception when setting the prototype.`)
}

type GetOwnProperties_Flag uint8

const (
	// use this bit-flag with the [Value.GetOwnProperties] method to include string based properties (including numeric properties).
	GetOwnProperties_StringFlag GetOwnProperties_Flag = C.JS_GPN_STRING_MASK
	// use this bit-flag with the [Value.GetOwnProperties] method to include symbol based properties.
	GetOwnProperties_SymbolFlag GetOwnProperties_Flag = C.JS_GPN_SYMBOL_MASK
	// use this bit-flag with the [Value.GetOwnProperties] method to private properties.
	GetOwnProperties_PrivateFlag GetOwnProperties_Flag = C.JS_GPN_PRIVATE_MASK
	// use this bit-flag with the [Value.GetOwnProperties] method to exclude non-enumerable properties (i.e. those not reported by `Object.keys()`).
	GetOwnProperties_EnumerableOnlyFlag GetOwnProperties_Flag = C.JS_GPN_ENUM_ONLY
)

// get the property [Atom]s of a javascript `Object`.
//
// use a combination of bit-flags (using the `|` (OR) bit operand) in the `flags` parameter to specify what _kind_ of properties to include.
// the available flags are:
// - [GetOwnProperties_StringFlag]: include string based properties (including numeric properties).
// - [GetOwnProperties_SymbolFlag]: include symbol based properties.
// - [GetOwnProperties_PrivateFlag]: include private properties.
// - [GetOwnProperties_EnumerableOnlyFlag]: exclude non-enumerable properties (i.e. those not reported by `Object.keys()`).
//
// @should-free
func (obj *Value) GetOwnProperties(flags GetOwnProperties_Flag) []*Atom {
	ctx := obj.ctx
	var first_result_ptr *C.JSPropertyEnum
	var size C.uint32_t
	// success is `-1` if an exception occurs (such as `obj` not actually being an `Object` type), or `0` when successful
	success := C.JS_GetOwnPropertyNames(ctx.ref, &first_result_ptr, &size, obj.ref, C.int(flags))
	if success < 0 {
		panic(`[Value.GetOwnProperties]: the provided value is not of "Object" type.`)
	}
	defer C.JS_FreePropertyEnum(ctx.ref, first_result_ptr, size)
	props_ref := unsafe.Slice(first_result_ptr, size)
	props := make([]*Atom, size)
	for i, prop_ref := range props_ref {
		// we must duplicate the atoms, because the `JS_FreePropertyEnum` frees each once, but we want the atoms to outlive this function.
		atom := (&Atom{ctx: ctx, ref: prop_ref.atom}).Dupe()
		props[i] = atom
	}
	return props
}

// get a javascript `Object`'s enumerable and non-enumerable string-based properties (analogous to `Object.getOwnPropertyNames(obj)` in javascript).
//
// this method is equivalent to calling the [Value.GetOwnProperties] method with the [GetOwnProperties_StringFlag] flag,
// and then converting the [Atom]s to `string`s, followed by freeing up the atoms afterwards.
func (obj *Value) GetOwnPropertyNames() []string {
	atoms := obj.GetOwnProperties(GetOwnProperties_StringFlag)
	names := make([]string, len(atoms))
	for i, atom := range atoms {
		names[i] = atom.ToString()
		defer atom.Free()
	}
	return names
}

// get a javascript `Object`'s enumerable and non-enumerable symbol-based properties (analogous to `Object.getOwnPropertySymbols(obj)` in javascript).
// it might sound weird that `symbol` based properties are enumerable, but they in fact are enumerable by default, despite not being returned by `Object.Keys(obj)` et al.
//
// this method is equivalent to calling the [Value.GetOwnProperties] method with the [GetOwnProperties_SymbolFlag] flag,
// and then converting the [Atom]s to `symbol`s (via [Atom.ToValue]), followed by freeing up the atoms afterwards.
//
// @should-free
func (obj *Value) GetOwnPropertySymbols() []*Value {
	atoms := obj.GetOwnProperties(GetOwnProperties_SymbolFlag)
	symbols := make([]*Value, len(atoms))
	for i, atom := range atoms {
		symbols[i] = atom.ToValue()
		defer atom.Free()
	}
	return symbols
}

// get a javascript `Object`'s enumerable and non-enumerable private properties.
// the returned values are of javascript [Value] types, because the could be either of `string`, `number`, or `symbol` type.
//
// @should-free
func (obj *Value) GetOwnPropertyPrivates() []*Value {
	atoms := obj.GetOwnProperties(GetOwnProperties_StringFlag | GetOwnProperties_SymbolFlag | GetOwnProperties_PrivateFlag)
	vals := make([]*Value, len(atoms))
	for i, atom := range atoms {
		vals[i] = atom.ToValue()
		defer atom.Free()
	}
	return vals
}

// TODO:
// func (obj *Value) GetOwnPropertyDescriptor() {
//
// }

// TODO:
// func (obj *Value) GetOwnPropertyDescriptors() {
//
// }

// get a javascript `Object`'s enumerable string-based properties (analogous to `Object.getOwnPropertyNames(obj)` in javascript).
//
// this method is equivalent to calling the [Value.GetOwnProperties] method with the [GetOwnProperties_StringFlag] and [GetOwnProperties_EnumerableOnlyFlag] flags,
// and then converting the [Atom]s to `string`s, followed by freeing up the atoms afterwards.
func (obj *Value) GetKeys() []string {
	atoms := obj.GetOwnProperties(GetOwnProperties_StringFlag | GetOwnProperties_EnumerableOnlyFlag)
	names := make([]string, len(atoms))
	for i, atom := range atoms {
		names[i] = atom.ToString()
		defer atom.Free()
	}
	return names
}

// get a javascript `Object`'s enumerable values (analogous to `Object.values(obj)` in javascript).
//
// @should-free
func (obj *Value) GetValues() []*Value {
	entries := obj.GetAtomicEntries()
	values := make([]*Value, len(entries))
	for i, entry := range entries {
		values[i] = entry.Value
		defer entry.Key.Free()
	}
	return values
}

type ObjectAtomicEntry struct {
	Key   *Atom
	Value *Value
}

type ObjectEntry struct {
	Key   string
	Value *Value
}

// get a javascript `Object`'s enumerable key-value pairs, with the keys being quickjs [Atom]s, rather than `strings`.
//
// @should-free
func (obj *Value) GetAtomicEntries() []ObjectAtomicEntry {
	atoms := obj.GetOwnProperties(GetOwnProperties_StringFlag | GetOwnProperties_EnumerableOnlyFlag)
	entries := make([]ObjectAtomicEntry, len(atoms))
	for i, atom := range atoms {
		entries[i].Key = atom
		entries[i].Value = obj.GetAtom(atom)
	}
	return entries
}

// get a javascript `Object`'s enumerable key-value pairs (analogous to `Object.entries(obj)` in javascript).
//
// @should-free
func (obj *Value) GetEntries() []ObjectEntry {
	atoms := obj.GetOwnProperties(GetOwnProperties_StringFlag | GetOwnProperties_EnumerableOnlyFlag)
	entries := make([]ObjectEntry, len(atoms))
	for i, atom := range atoms {
		entries[i].Key = atom.ToString()
		entries[i].Value = obj.GetAtom(atom)
		defer atom.Free()
	}
	return entries
}
