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

type Context struct {
	rt  *Runtime
	ref *C.JSContext
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
