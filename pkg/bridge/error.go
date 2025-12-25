// this file contains a wrapper for javascript's `Error` class instances.

package bridge

/*
#include "./include0_quickjs.h"
*/
import "C"
import fmt "fmt"

// represents a quickjs error formatted for go, in addition to also implementing the `error` go interface.
type Error struct {
	Name    string // the name of the js-error class, such as `SyntaxError`, `TypeError`, etc...
	Message string // the error message.
	Cause   string // the cause behind the error.
	Stack   string // the stack trace before the error.
}

// prints the error message as a string (to implement the `error` interface).
func (err *Error) Error() string {
	message := fmt.Sprintf("[%s]: %s", err.Name, err.Message)
	if err.Cause == "" {
		return message
	}
	return fmt.Sprintf("%s (cause: %s)", message, err.Cause)
}

// spawn a new javascript exception value (which is different from an `Error`).
//
// note that it does not need to be freed afterwards.
func (ctx *Context) NewException() *Value {
	return &Value{ctx: ctx, ref: C.JS_EXCEPTION}
}

// create a new javascript error with a given error message.
//
// you should make sure that `error` is **not** a `nil`!
//
// @should-free
func (ctx *Context) NewError(err error) *Value {
	val := &Value{ctx: ctx, ref: C.JS_NewError(ctx.ref)}
	val.Set("message", ctx.NewString(err.Error()))
	return val
}

// if the js-value is an `Error`, a go `error` will be returned (containing its internal message), otherwise you will receive a `nil`.
func (val *Value) ToError() *Error {
	if !val.IsError() {
		return nil
	}
	err := &Error{}
	name := val.Get("name")
	message := val.Get("message")
	cause := val.Get("cause")
	stack := val.Get("stack")
	defer name.Free()
	defer message.Free()
	defer cause.Free()
	defer stack.Free()
	if !name.IsUndefined() {
		err.Name = name.ToString()
	}
	if !message.IsUndefined() {
		err.Message = message.ToString()
	}
	if !cause.IsUndefined() {
		err.Cause = cause.ToString()
	}
	if !stack.IsUndefined() {
		err.Stack = stack.ToString()
	}
	return err
}
