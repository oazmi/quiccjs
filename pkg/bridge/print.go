// this file contains methods for printing the string representation of javascript [Value]s.
//
// TODO: should I perhaps rename this file to `debug.go`, and dedicate it to code related to _inspecting_ javascript values?

package bridge

/*
#include "./include0_quickjs.h"

// forward declaration of the of the buffer-writing callback function, otherwise the compiler won't discover it.
JSPrintValueWrite printValueWriteFn;

// moreover, since `JSPrintValueWrite` uses a `const char* buf` in its parameters, we cannot use `buf *C.char` in its place in the go-definition,
// because cgo will complain about the two not being interchangeable. thus we must create a typedef for `const char*` here in the c-preamble,
// as there is no other way to represent it in the go-code. I found the answer to this problem on stackexchange: "https://stackoverflow.com/a/63296856".
typedef const char const_char_t;
*/
import "C"
import (
	bytes "bytes"
	cgo "runtime/cgo"
	unsafe "unsafe"
)

type printBuffer = bytes.Buffer

//export printValueWriteFn
func printValueWriteFn(opaque_buf_ptr unsafe.Pointer, buf_first_byte_ptr *C.const_char_t, buf_len C.size_t) {
	buf_handle := cgo.Handle(opaque_buf_ptr)
	buf_ptr := buf_handle.Value().(*printBuffer)
	buf_ptr.Write(unsafe.Slice((*byte)(unsafe.Pointer(buf_first_byte_ptr)), int(buf_len)))
}

// get the _printed_ string representation of a javascript [Value].
//
// the printed string differs from the [Value.ToString] representation, because here,
// you will receive the string equivalent of what you would get from a `console.log(val)` print.
// but on the other hand, [Value.ToString] will return what the value's `String()` javascript-method returns.
//
// for instance, here's how the javascript array object `[1, 2, 3, 4]` will be represented by each of the two methods:
// - [Value.ToString]: `[object Object]`
// - [Value.PrintString]: `[1, 2, 3, 4]`
func (val *Value) PrintString() string {
	buf_ptr := &printBuffer{}
	// a specialized `Handle` must be used when transporting a pointer unsafely and opaquely,
	// so that go will guarantee that the receiver of the opaque void pointer will be able to reconstruct the original type via casting
	// (in addition to also not moving around the memory region of the original object until the handle has been deleted).
	// you may wonder why we delete the handle upon exiting this function,
	// rather than inside of the `printValueWriteFn` function once it has been executed;
	// and the reason for that is because `JS_PrintValue` does not guaranteen performing just a single call;
	// it _may_ perform multiple calls to the `printValueWriteFn` function.
	// however, it is guaranteed to be synchronous and sequential, thus this function won't be exiting until the string has been prepared.
	buf_handle := cgo.NewHandle(buf_ptr)
	defer buf_handle.Delete()
	// setting the `options` parameter to `nil` gives us the default options.
	// TODO: in the future, attach the printing `options` to your `ctx`, and then pass it here.
	C.JS_PrintValue(val.ctx.ref, &C.printValueWriteFn, unsafe.Pointer(buf_handle), val.ref, nil)
	return buf_ptr.String()
}
