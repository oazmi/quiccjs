// this file contains a wrapper for javascript `Function`s.

package bridge

/*
#include "./include0_quickjs.h"
*/
import "C"
import (
	runtime "runtime"
)

// bind a javascript function to some default arguments.
//
// the signature of this method mimics javascript's `Function.prototype.bind(thisArg, ...args)` method.
// except, you must always provide the first argument.
// in case there is no `this` object that needs to be referenced by your function, simply set the `this` argument to `nil`.
func (fun *Value) Bind(this *Value, default_args ...*Value) func(args ...*Value) *Value {
	ctx := fun.ctx
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
		result_ref := C.JS_Call(ctx.ref, fun.ref, this.ref, C.int(default_args_len), first_js_arg_ptr)
		return &Value{ctx: ctx, ref: result_ref}
	}
}

// execute a javascript `Function` with the given `this` object and the given `args`.
//
// the signature of this method mimics javascript's `Function.prototype.call(thisArg, ...args)` method.
// except, you must always provide the first argument.
// in case there is no `this` object that needs to be referenced by your function, simply set the `this` argument to `nil`.
func (fun *Value) Call(this *Value, args ...*Value) *Value {
	ctx := fun.ctx
	var this_ref C.JSValue
	if this == nil {
		this_ref = C.JS_UNDEFINED
	} else {
		this_ref = this.ref
	}

	// inlined logic of `Context.valuesToCValues()`, because we don't want the returned slice to be allocated on the heap instead of the stack.
	// a slice "escapes" to the heap when its lifetime extends beyond the function that created it.
	// inline start
	args_len := C.int(len(args))
	var stack_allocated_args [max_stack_js_args]C.JSValue
	var heap_allocated_args []C.JSValue
	var first_js_arg_ptr *C.JSValue = nil
	if args_len == 0 {
	} else if args_len <= max_stack_js_args {
		for i, js_arg := range args {
			stack_allocated_args[i] = js_arg.ref
		}
		first_js_arg_ptr = &stack_allocated_args[0]
	} else {
		heap_allocated_args = make([]C.JSValue, args_len)
		for i, js_arg := range args {
			heap_allocated_args[i] = js_arg.ref
		}
		first_js_arg_ptr = &heap_allocated_args[0]
	}
	// inline end

	result_ref := C.JS_Call(ctx.ref, fun.ref, this_ref, args_len, first_js_arg_ptr)
	// tell the compiler that the allocated slices should live up to this point at least,
	// so that go does not free them while quick js is using them.
	runtime.KeepAlive(stack_allocated_args)
	runtime.KeepAlive(heap_allocated_args)
	return &Value{ctx: ctx, ref: result_ref}
}

// execute a class's constructor with the given arguments to produce a class instance.
func (cls *Value) CallConstructor(args ...*Value) *Value {
	ctx := cls.ctx

	// inlined logic of `Context.valuesToCValues()`, because we don't want the returned slice to be allocated on the heap instead of the stack.
	// a slice "escapes" to the heap when its lifetime extends beyond the function that created it.
	// inline start
	args_len := C.int(len(args))
	var stack_allocated_args [max_stack_js_args]C.JSValue
	var heap_allocated_args []C.JSValue
	var first_js_arg_ptr *C.JSValue = nil
	if args_len == 0 {
	} else if args_len <= max_stack_js_args {
		for i, js_arg := range args {
			stack_allocated_args[i] = js_arg.ref
		}
		first_js_arg_ptr = &stack_allocated_args[0]
	} else {
		heap_allocated_args = make([]C.JSValue, args_len)
		for i, js_arg := range args {
			heap_allocated_args[i] = js_arg.ref
		}
		first_js_arg_ptr = &heap_allocated_args[0]
	}
	// inline end

	result_ref := C.JS_CallConstructor(ctx.ref, cls.ref, args_len, first_js_arg_ptr)
	// tell the compiler that the allocated slices should live up to this point at least,
	// so that go does not free them while quick js is using them.
	runtime.KeepAlive(stack_allocated_args)
	runtime.KeepAlive(heap_allocated_args)
	return &Value{ctx: ctx, ref: result_ref}
}
