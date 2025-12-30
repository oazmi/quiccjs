// this file contains a wrapper for quickjs _atom_s, which are an efficient representation of property strings.
// in some sense, their purpose resembles that of served by javascript `Symbol`s.

package bridge

/*
#include "./include0_quickjs.h"
*/
import "C"
import (
	fmt "fmt"
	unsafe "unsafe"
)

// an atom is an efficient unique `int32` representation of property strings, internal to quickjs.
type Atom struct {
	ctx *Context
	ref C.JSAtom
}

// free up an [Atom].
func (atom *Atom) Free() {
	C.JS_FreeAtom(atom.ctx.ref, atom.ref)
}

// decrements the reference count of a quickjs [Atom] object _when_ its [Context] exits/frees up (i.e. when [Context.Free] is called).
//
// this is opposed to freeing up the value _immediately_ via [Atom.Free].
// it is intended for cached value properties, such as `length` and `size`.
func (atom *Atom) FreeOnExit() {
	atom.ctx.atomFreeupList = append(atom.ctx.atomFreeupList, atom)
}

// increments the reference count of a quickjs [Atom] object, and duplicates its [Atom] wrapper.
//
// it is quite rare for this method to be used outside of the internal library logic,
// so be sure that your logic is sound if you're using this method.
//
// to decrement the reference count, use the [Atom.Free] method.
func (atom *Atom) Dupe() *Atom {
	// the operation only takes place on non-nil atom and contexts.
	if atom == nil || atom.ctx == nil {
		return nil
	}
	return &Atom{ctx: atom.ctx, ref: C.JS_DupAtom(atom.ctx.ref, atom.ref)}
}

// create a new quickjs atom from a given go-string.
// don't forget to free up atoms after you are done using them.
//
// @should-free
func (ctx *Context) NewAtom(v string) *Atom {
	cstr_ptr := C.CString(v)
	defer C.free(unsafe.Pointer(cstr_ptr))
	return &Atom{ctx: ctx, ref: C.JS_NewAtom(ctx.ref, cstr_ptr)}
}

// create a new quickjs atom from a given numeric property index.
// don't forget to free up atoms after you are done using them.
//
// @should-free
func (ctx *Context) NewAtomIdx(idx uint32) *Atom {
	return &Atom{ctx: ctx, ref: C.JS_NewAtomUInt32(ctx.ref, C.uint32_t(idx))}
}

// returns the string representation of the atomic property.
func (atom *Atom) ToString() string {
	js_string := atom.ToValue()
	defer js_string.Free()
	return js_string.ToString()
}

// returns the javascript [Value] representation of the atomic property.
// it can either be a `number`-based, a `string`-based, or a `symbol`-based javascript value.
//
// @should-free
func (atom *Atom) ToValue() *Value {
	return &Value{ctx: atom.ctx, ref: C.JS_AtomToValue(atom.ctx.ref, atom.ref)}
}

// converts the primitive javascript [Value] property key to its of the [Atom]ic representation.
// this might be the only way to convert a javascript `symbol` to an [Atom] to use for property acquisition.
//
// @should-free
func (val *Value) ToAtom() *Atom {
	atom_ref := C.JS_ValueToAtom(val.ctx.ref, val.ref)
	if atom_ref == C.JS_ATOM_NULL {
		panic("[Value.ToAtom]: failed to convert the provided value to an atom, possibly because the value is not a valid property key (i.e. neither a number, nor a string, nor a symbol).")
	}
	return &Atom{ctx: val.ctx, ref: atom_ref}
}

// set a javascript `Object`'s atomic property `prop_atom` to a certain value `val`.
//
// this operation is faster than the string-based [Value.Set] method,
// but, **you**, the user, will have to bear the overhead of creating the `prop_atom`,
// and also bear the burden of freeing it once you've made all the necessary changes related to this atomic property.
//
// @ownership-transfer
func (obj *Value) SetAtom(prop_atom *Atom, val *Value) {
	success := C.JS_SetProperty(obj.ctx.ref, obj.ref, prop_atom.ref, val.ref)
	// success is either `-1` (exception), `0` (false), or `1` (true).
	if success < 0 {
		panic(fmt.Sprintf(`[Object.SetAtom]: setting the value of the atomic property "%s" resulted in an exception. your value may not be an "Object".`, prop_atom.ToString()))
	}
}

// get the value of a javascript `Object`'s atomic property `prop_atom`.
//
// this operation is faster than the string-based [Value.Get] method,
// but, **you**, the user, will have to bear the overhead of creating the `prop_atom`,
// and also bear the burden of freeing it once you've made all the necessary changes related to this atomic property.
//
// @should-free
func (obj *Value) GetAtom(prop_atom *Atom) *Value {
	return &Value{ctx: obj.ctx, ref: C.JS_GetProperty(obj.ctx.ref, obj.ref, prop_atom.ref)}
}

// dictates whether or not an `Object` has a certain atomic property `prop_atom`.
//
// this operation is faster than the string-based [Value.Has] method,
// but, **you**, the user, will have to bear the overhead of creating the `prop_atom`,
// and also bear the burden of freeing it once you've made all the necessary changes related to this atomic property.
func (obj *Value) HasAtom(prop_atom *Atom) bool {
	success := C.JS_HasProperty(obj.ctx.ref, obj.ref, prop_atom.ref)
	if success >= 0 {
		return (success == 1)
	}
	panic(fmt.Sprintf(`[Object.HasAtom]: checking for the atomic property "%s" resulted in an exception. your value may not be an "Object".`, prop_atom.ToString()))
}

// delete/remove javascript `Object`'s atomic property `prop_atom`, and have it freed (the property field will become _uninitialized_ afterwards).
//
// a `true` returned value indicates that the `prop_atom` property had been initialized prior to being deleted,
// while a `false` would indicate that the `prop_atom` property had never been initialized before.
//
// this operation is faster than the string-based [Value.Delete] method,
// but, **you**, the user, will have to bear the overhead of creating the `prop_atom`,
// and also bear the burden of freeing it once you've made all the necessary changes related to this atomic property.
func (obj *Value) DeleteAtom(prop_atom *Atom) bool {
	success := C.JS_DeleteProperty(obj.ctx.ref, obj.ref, prop_atom.ref, 1)
	if success >= 0 {
		return (success == 1)
	}
	panic(fmt.Sprintf(`[Object.DeleteAtom]: deleting the atomic property "%s" resulted in an exception. your value may not be an "Object".`, prop_atom.ToString()))
}
