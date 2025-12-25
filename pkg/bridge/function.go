// this file contains a wrapper for javascript `Function`s.

package bridge

/*
#include "./include0_quickjs.h"
*/
import "C"

// bind a javascript function to some default arguments.
//
// the signature of this method mimics javascript's `Function.prototype.bind(thisArg, ...args)` method.
// except, you must always provide the first argument.
// in case there is no `this` object that needs to be referenced by your function, simply set the `this` argument to `nil`.
func (fn *Value) Bind(this *Value, default_args ...*Value) func(args ...*Value) *Value {
	ctx := fn.ctx
	if this == nil {
		this = ctx.NewUndefined()
	}
	default_args_len := len(default_args)
	// we pre-allocate the slice below to ensure a contiguous block of memory is allocated to the slice,
	// otherwise quickjs will not be able to index elements that are not in the continuous region (i.e. memory corruption).
	js_default_args := make([]C.JSValue, default_args_len)
	for i, js_arg := range default_args {
		js_default_args[i] = js_arg.ref
	}
	return func(args ...*Value) *Value {
		args_len := len(args)
		js_args := make([]C.JSValue, default_args_len+args_len)
		copy(js_args, js_default_args) // this operation is fast when the slice is contiguous.
		for i, js_arg := range args {
			js_args[default_args_len+i] = js_arg.ref
		}
		var first_js_arg_ptr *C.JSValue = nil
		if (default_args_len + args_len) > 0 {
			first_js_arg_ptr = &js_args[0]
		}
		result_ref := C.JS_Call(ctx.ref, fn.ref, this.ref, C.int(default_args_len), first_js_arg_ptr)
		return &Value{ctx: ctx, ref: result_ref}
	}
}

// execute a javascript `Function` with the given `this` object and the given `args`.
//
// the signature of this method mimics javascript's `Function.prototype.call(thisArg, ...args)` method.
// except, you must always provide the first argument.
// in case there is no `this` object that needs to be referenced by your function, simply set the `this` argument to `nil`.
func (fn *Value) Call(this *Value, args ...*Value) *Value {
	ctx := fn.ctx
	if this == nil {
		this = ctx.NewUndefined()
	}
	args_len := len(args)
	// we pre-allocate the slice below to ensure a contiguous block of memory is allocated to the slice,
	// otherwise quickjs will not be able to index elements that are not in the continuous region (i.e. memory corruption).
	js_args := make([]C.JSValue, args_len)
	for i, js_arg := range args {
		js_args[i] = js_arg.ref
	}
	var first_js_arg_ptr *C.JSValue = nil
	if args_len > 0 {
		first_js_arg_ptr = &js_args[0]
	}
	result_ref := C.JS_Call(ctx.ref, fn.ref, this.ref, C.int(args_len), first_js_arg_ptr)
	return &Value{ctx: ctx, ref: result_ref}
}

// execute a class's constructor with the given arguments to produce a class instance.
func (cls *Value) CallConstructor(args ...*Value) *Value {
	ctx := cls.ctx
	args_len := len(args)
	// we pre-allocate the slice below to ensure a contiguous block of memory is allocated to the slice,
	// otherwise quickjs will not be able to index elements that are not in the continuous region (i.e. memory corruption).
	js_args := make([]C.JSValue, args_len)
	var first_js_arg_ptr *C.JSValue = nil
	for i, js_arg := range args {
		js_args[i] = js_arg.ref
	}
	if args_len > 0 {
		first_js_arg_ptr = &js_args[0]
	}
	result_ref := C.JS_CallConstructor(ctx.ref, cls.ref, C.int(args_len), first_js_arg_ptr)
	return &Value{ctx: ctx, ref: result_ref}
}
