// this file contains utility helper functions and type definitions.

package bridge

/*
#include "./include0_quickjs.h"
*/
import "C"
import unsafe "unsafe"

// convert a c-slice of `JSValue`s to a slice of [Value]s.
// this is a method of the [Context] because the [Value]s will need to hold a reference to their context,
// thus we'll need to copy over its reference to each [Value].
func (ctx *Context) cValuesToValues(args_len C.int, first_arg_ptr *C.JSValue) []*Value {
	refs := unsafe.Slice(first_arg_ptr, args_len)
	vals := make([]*Value, args_len)
	for i, ref := range refs {
		vals[i] = &Value{ctx: ctx, ref: ref}
	}
	return vals
}

// convert a slice of [Value]s to a c-slice of `JSValue`s,
// with the pointer to the first `JSValue` returned, in addition to the entire slice's length as a 2-tuple.
//
// this method does not involve the [Context], but I've made it such, for organizational purposes.
//
// when the input `args` slice is empty, the returned value will be `(0, nil)`.
// so make sure you always observe the length first before utilizing the pointer to the first `JSValue`.
//
// returns: `(args_length, first_js_arg_pointer)`
func (ctx *Context) valuesToCValues(args []*Value) (C.int, *C.JSValue) {
	args_len := C.int(len(args))
	// we pre-allocate the slice below to ensure a contiguous block of memory is allocated to the slice,
	// otherwise quickjs will not be able to index elements that are not in the continuous region (i.e. memory corruption).
	// TODO: does quickjs free the memory allocated by us here, or does it not free it after executing the function?
	js_args := make([]C.JSValue, args_len)
	var first_js_arg_ptr *C.JSValue = nil
	for i, js_arg := range args {
		js_args[i] = js_arg.ref
	}
	if args_len > 0 {
		first_js_arg_ptr = &js_args[0]
	}
	return args_len, first_js_arg_ptr
}
