package bridge

/*
#include "./include0_quickjs.h"
*/
import "C"
import (
	errors "errors"
	fmt "fmt"
	runtime "runtime"
	unsafe "unsafe"
)

type Context struct {
	rt         *Runtime
	ref        *C.JSContext
	valueCache contextValueCache
	freeUpList []*Value
}

type contextValueCache struct {
	// miscellaneous
	globalThis *Value
	date       *Value
	// collections
	array   *Value
	hashMap *Value
	hashSet *Value
	// typed arrays and buffers
	arrayBuffer       *Value
	uint8Array        *Value
	uint16Array       *Value
	uint32Array       *Value
	int8Array         *Value
	int16Array        *Value
	int32Array        *Value
	uint8ClampedArray *Value
	bigUint64Array    *Value
	bigInt64Array     *Value
	float16Array      *Value
	float32Array      *Value
	float64Array      *Value
}

func (rt *Runtime) NewContext() *Context {
	if rt.ref == nil {
		return nil
	}
	ctx := &Context{
		ref:        C.JS_NewContext(rt.ref),
		rt:         rt,
		valueCache: contextValueCache{},
		freeUpList: []*Value{},
	}
	if ctx.ref == nil {
		return nil
	}
	ctx.injectValueCache()
	// TODO: I'm unsure if we should be cleaning it up automatically, or if it should be the end user's responsibility.
	runtime.AddCleanup(ctx, (*Context).Free, nil)
	return ctx
}

func (ctx *Context) Free() {
	if ctx.ref != nil {
		for _, val := range ctx.freeUpList {
			val.Free()
		}
		C.JS_FreeContext(ctx.ref)
		ctx.ref = nil
	}
}

func (ctx *Context) injectValueCache() {
	global_this := &Value{ctx: ctx, ref: C.JS_GetGlobalObject(ctx.ref)}
	ctx.valueCache.globalThis = global_this
	global_this.FreeOnExit()

	get_obj := func(object_name string) *Value {
		js_obj := global_this.Get(object_name)
		if js_obj.IsObject() {
			js_obj.FreeOnExit()
			return js_obj
		}
		panic(fmt.Sprintf(`[Context.injectValueCache]: missing a global class from the js-context: "%s".`, object_name))
	}
	// collections
	ctx.valueCache.array = get_obj("Array")
	ctx.valueCache.hashMap = get_obj("Map")
	ctx.valueCache.hashSet = get_obj("Set")
	// typed arrays and buffers
	ctx.valueCache.arrayBuffer = get_obj("ArrayBuffer")
	ctx.valueCache.uint8Array = get_obj("Uint8Array")
	ctx.valueCache.uint16Array = get_obj("Uint16Array")
	ctx.valueCache.uint32Array = get_obj("Uint32Array")
	ctx.valueCache.int8Array = get_obj("Int8Array")
	ctx.valueCache.int16Array = get_obj("Int16Array")
	ctx.valueCache.int32Array = get_obj("Int32Array")
	ctx.valueCache.uint8ClampedArray = get_obj("Uint8ClampedArray")
	ctx.valueCache.bigUint64Array = get_obj("BigUint64Array")
	ctx.valueCache.bigInt64Array = get_obj("BigInt64Array")
	ctx.valueCache.float16Array = get_obj("Float16Array")
	ctx.valueCache.float32Array = get_obj("Float32Array")
	ctx.valueCache.float64Array = get_obj("Float64Array")
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
