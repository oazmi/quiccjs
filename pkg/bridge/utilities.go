// this file contains utility helper functions and type definitions.

package bridge

/*
#include "./include0_quickjs.h"
*/
import "C"
import unsafe "unsafe"

// any function with these may args or less will have its `JSValue` allocated on the stack rather than the heap to speedup copying and transferring.
//
// since each `JSValue` is 16-bytes large (8 for pointer or double, 8 for js-type-tag),
// our conservative choice of `4` would mean that each recursion call would add 64-bytes to the stack,
// (even when the heap path is used when the number of args exceeds `4`).
// this is _almost_ safe on most systems (for instance, esp32 has a stack size of 4kb, and other architectures often have 1mb of stack size minimum).
const max_stack_js_args = 4

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
//
// > [!warning]
// > generally, you should inline this function, because the allocated memory is not guaranteed to stay alive after we return the pointer,
// > and it gets passed to a cgo function that might call it later (asynchronously).
// > moreover, the stack allocation automatically gets promoted to a heap allocation when the slice's lifetime escapes the function;
// > which is not something that we want. but it can only be avoided by inlining the function.
func (ctx *Context) valuesToCValues(args []*Value) (C.int, *C.JSValue) {
	args_len := C.int(len(args))
	// fastest zero-arg path.
	if args_len == 0 {
		return 0, nil
	}
	// fast 4-max-arg path, achieved by stack allocation.
	if args_len <= max_stack_js_args {
		var stack_allocated_args [max_stack_js_args]C.JSValue
		for i, js_arg := range args {
			stack_allocated_args[i] = js_arg.ref
		}
		return args_len, &stack_allocated_args[0]
	}
	// slower var-arg path, via heap allocation.
	// we pre-allocate the slice below to ensure a contiguous block of memory is allocated to the slice,
	// otherwise quickjs will not be able to index elements that are not in the continuous region (i.e. memory corruption).
	// TODO: does quickjs free the memory allocated by us here, or does it not free it after executing the function?
	heap_allocated_args := make([]C.JSValue, args_len)
	for i, js_arg := range args {
		heap_allocated_args[i] = js_arg.ref
	}
	return args_len, &heap_allocated_args[0]
}
