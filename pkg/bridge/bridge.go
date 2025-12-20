package bridge

/*
#include "./include0_quickjs.h"
*/
import "C"
import (
	errors "errors"
	runtime "runtime"
	unsafe "unsafe"
)

type Runtime struct {
	ref *C.JSRuntime
}

type Context struct {
	rt  *Runtime
	ref *C.JSContext
}

type Value struct {
	ctx *Context
	ref C.JSValue
}

func NewRuntime() *Runtime {
	rt := &Runtime{
		ref: C.JS_NewRuntime(),
	}
	if rt.ref == nil {
		return nil
	}
	// TODO: I'm unsure if we should be cleaning it up automatically, or if it should be the end user's responsibility.
	runtime.AddCleanup(rt, (*Runtime).Free, nil)
	return rt
}

func (rt *Runtime) Free() {
	if rt.ref != nil {
		C.JS_FreeRuntime(rt.ref)
		rt.ref = nil
	}
}

func (rt *Runtime) NewContext() *Context {
	if rt.ref == nil {
		return nil
	}
	ctx := &Context{
		ref: C.JS_NewContext(rt.ref),
		rt:  rt,
	}
	if ctx.ref == nil {
		return nil
	}
	// TODO: I'm unsure if we should be cleaning it up automatically, or if it should be the end user's responsibility.
	runtime.AddCleanup(ctx, (*Context).Free, nil)
	return ctx
}

func (ctx *Context) Free() {
	if ctx.ref != nil {
		C.JS_FreeContext(ctx.ref)
		ctx.ref = nil
	}
}

type js_EVAL_TYPE int // `0` or `1`

const (
	js_EVAL_TYPE_GLOBAL js_EVAL_TYPE = C.JS_EVAL_TYPE_GLOBAL // 0
	js_EVAL_TYPE_ASYNC  js_EVAL_TYPE = C.JS_EVAL_TYPE_GLOBAL | C.JS_EVAL_FLAG_ASYNC
)

func (ctx *Context) Eval(code string) (*Value, error) {
	return ctx.evalBase(code, js_EVAL_TYPE_GLOBAL)
}

func (ctx *Context) EvalAsync(code string) (*Value, error) {
	return ctx.evalBase(code, js_EVAL_TYPE_ASYNC)
}

func (ctx *Context) evalBase(code string, eval_type js_EVAL_TYPE) (*Value, error) {
	if ctx.ref == nil {
		return nil, errors.New("context is nil")
	}
	c_code := C.CString(code)
	c_filename := C.CString("<eval>") // I don't think it's possible to perform an eval without a file name.
	c_code_len := C.size_t(len(code)) // this is not `len(code) + 1` because the terminating null character must not be included in the code.
	c_eval_flag := C.int(eval_type)   // `JS_EVAL_TYPE_GLOBAL = 0` and `JS_EVAL_TYPE_MODULE = 1`
	defer C.free(unsafe.Pointer(c_code))
	defer C.free(unsafe.Pointer(c_filename))

	result := C.JS_Eval(ctx.ref, c_code, c_code_len, c_filename, c_eval_flag)
	if C.JS_IsException(result) != 0 {
		val := Value{ref: C.JS_GetException(ctx.ref), ctx: ctx}
		defer val.Free()
		println(val.ToString())
		return nil, errors.New("JS exception: " + val.ToString())
	}
	val := Value{ref: result, ctx: ctx}
	return &val, nil
}

// returns the `int64` value of the value.
func (val *Value) ToInt64() int64 {
	cval := C.int64_t(55)
	println(&cval, cval)
	println(C.JS_ToInt64(val.ctx.ref, &cval, val.ref))
	println(&cval, cval)
	return int64(cval)
}

// returns the `string` representation of a value.
func (val *Value) ToString() string {
	ptr := C.JS_ToCString(val.ctx.ref, val.ref)
	defer C.JS_FreeCString(val.ctx.ref, ptr)
	return C.GoString(ptr)
}

func (v *Value) Free() {
	if v.ctx == nil {
		return // no context or undefined value, nothing to free
	}
	C.JS_FreeValue(v.ctx.ref, v.ref)
}
