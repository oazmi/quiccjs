package bridge

/*
#include "./include0_quickjs.h"
*/
import "C"
import (
	runtime "runtime"
)

type Runtime struct {
	ref *C.JSRuntime
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
