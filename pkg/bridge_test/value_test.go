// this file contains tests for `runtime.go` file under the [bridge] package.

package bridge_test

import (
	math "math"
	big "math/big"
	testing "testing"

	js "github.com/oazmi/quiccjs/pkg/bridge"
)

func TestValue_Literals(t *testing.T) {
	rt := js.NewRuntime()
	defer rt.Free()
	ctx := rt.NewContext()
	defer ctx.Free()

	type testCase struct {
		name      string
		createFn  func() *js.Value
		checkType func(*js.Value) bool
		checkVal  func(*js.Value) bool
	}

	tests := []testCase{{
		name:      "Null",
		createFn:  ctx.NewNull,
		checkType: (*js.Value).IsNull,
		checkVal:  func(v *js.Value) bool { return true },
	}, {
		name:      "Undefined",
		createFn:  ctx.NewUndefined,
		checkType: (*js.Value).IsUndefined,
		checkVal:  func(v *js.Value) bool { return true },
	}, {
		name:      "Uninitialized",
		createFn:  ctx.NewUninitialized,
		checkType: (*js.Value).IsUninitialized,
		checkVal:  func(v *js.Value) bool { return true },
	}, {
		name:      "True",
		createFn:  func() *js.Value { return ctx.NewBool(true) },
		checkType: (*js.Value).IsBool,
		checkVal:  func(v *js.Value) bool { return v.ToBool() == true },
	}, {
		name:      "False",
		createFn:  func() *js.Value { return ctx.NewBool(false) },
		checkType: (*js.Value).IsBool,
		checkVal:  func(v *js.Value) bool { return v.ToBool() == false },
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val := tc.createFn()
			if !tc.checkType(val) {
				t.Errorf(`[type check ]: failed for: "%s"`, tc.name)
			}
			if !tc.checkVal(val) {
				t.Errorf(`[value check]: failed for: "%s"`, tc.name)
			}
		})
	}
}

func TestValue_Numbers(t *testing.T) {
	rt := js.NewRuntime()
	defer rt.Free()
	ctx := rt.NewContext()
	defer ctx.Free()

	test_number := int64(-110011)
	test_name := "Int32"
	// in all of the cases, we don't free up the js `val` because they _should not_ require to be freed.
	t.Run(test_name, func(t *testing.T) {
		val := ctx.NewInt32(int32(test_number))
		if !val.IsNumber() {
			t.Errorf(`[type check ]: expected js-type "number" for test: "%s"`, test_name)
		}
		if got := val.ToInt32(); got != int32(test_number) {
			t.Errorf(`[value check]: expected value: "%d", got: "%d", for test: "%s"`, test_number, got, test_name)
		}
	})

	test_name = "Uint32"
	t.Run(test_name, func(t *testing.T) {
		val := ctx.NewUint32(uint32(test_number))
		if !val.IsNumber() {
			t.Errorf(`[type check ]: expected js-type "number" for test: "%s"`, test_name)
		}
		if got := val.ToUint32(); got != uint32(test_number) {
			t.Errorf(`[value check]: expected value: "%d", got: "%d", for test: "%s"`, uint32(test_number), got, test_name)
		}
	})

	test_name = "Int64"
	test_number = int64(-9007199254740991) // `- Number.MAX_SAFE_INTEGER`, i.e. 53-bits are set to `true`.
	t.Run(test_name, func(t *testing.T) {
		val := ctx.NewInt64(int64(test_number))
		if !val.IsNumber() {
			t.Errorf(`[type check ]: expected js-type "number" for test: "%s"`, test_name)
		}
		if got := val.ToInt64(); got != int64(test_number) {
			t.Errorf(`[value check]: expected value: "%d", got: "%d", for test: "%s"`, int64(test_number), got, test_name)
		}
	})

	test_name = "Int64 - imprecise"
	test_number = int64(18014398509481983) // 54-bits are set to `true`, which should lead to imprecision in javascript.
	t.Run(test_name, func(t *testing.T) {
		expected_imprecise_int := test_number + 1
		val := ctx.NewInt64(int64(test_number))
		if got := val.ToInt64(); got != expected_imprecise_int {
			t.Errorf(`[value check]: expected imprecise conversion of: "%d", to: "%d", but got: "%d" for test: "%s"`, int64(test_number), expected_imprecise_int, got, test_name)
		}
	})

	test_name = "Float64"
	test_float := float64(123.456)
	t.Run(test_name, func(t *testing.T) {
		val := ctx.NewFloat64(test_float)
		if !val.IsNumber() {
			t.Errorf(`[type check ]: expected js-type "number" for test: "%s"`, test_name)
		}
		if got := val.ToFloat64(); got != test_float {
			t.Errorf(`[value check]: expected value: "%f", got: "%f", for test: "%s"`, test_float, got, test_name)
		}
	})

	test_name = "Float64 - max value"
	test_float = 1.7976931348623157 * math.Pow10(308) // largest double float.
	t.Run(test_name, func(t *testing.T) {
		val := ctx.NewFloat64(test_float)
		if got := val.ToFloat64(); got != test_float {
			t.Errorf(`[value check]: expected value: "%f", got: "%f", for test: "%s"`, test_float, got, test_name)
		}
	})

	test_name = "Float64 - min value"
	test_float = 2.2250738585072014 * math.Pow10(-308) // small double float.
	t.Run(test_name, func(t *testing.T) {
		val := ctx.NewFloat64(test_float)
		if got := val.ToFloat64(); got != test_float {
			t.Errorf(`[value check]: expected value: "%f", got: "%f", for test: "%s"`, test_float, got, test_name)
		}
	})

	test_name = "Float64 - infinities"
	t.Run(test_name, func(t *testing.T) {
		test_float = math.Inf(0)
		val_inf := ctx.NewFloat64(test_float)
		if got := val_inf.ToFloat64(); got != test_float {
			t.Errorf(`[value check]: expected value: "%f", got: "%f", for test: "%s"`, test_float, got, test_name)
		}

		test_float = math.Inf(-1)
		val_neg_inf := ctx.NewFloat64(test_float)
		if got := val_neg_inf.ToFloat64(); got != test_float {
			t.Errorf(`[value check]: expected value: "%f", got: "%f", for test: "%s"`, test_float, got, test_name)
		}
	})
}

func TestValue_BigInts(t *testing.T) {
	rt := js.NewRuntime()
	defer rt.Free()
	ctx := rt.NewContext()
	defer ctx.Free()

	test_name := "BigInt64"
	t.Run(test_name, func(t *testing.T) {
		test_int := int64(-18014398509481983) // 54-bits are set to `true` (greater than `Number.MAX_SAFE_INTEGER`).
		val := ctx.NewBigInt64(test_int)
		defer val.Free()
		if !val.IsBigInt() {
			t.Errorf(`[type check ]: expected js-type "bigint" for test: "%s"`, test_name)
		}
		if got := val.ToBigInt64(); got != test_int {
			t.Errorf(`[value check]: expected value: "%d", got: "%d", for test: "%s"`, test_int, got, test_name)
		}
	})

	test_name = "BigUint64"
	t.Run(test_name, func(t *testing.T) {
		test_int := uint64(18446744073709551615) // largest uint64.
		val := ctx.NewBigUint64(test_int)
		defer val.Free()
		if !val.IsBigInt() {
			t.Errorf(`[type check ]: expected js-type "bigint" for test: "%s"`, test_name)
		}
		if got := val.ToBigInt().Uint64(); got != test_int {
			t.Errorf(`[value check]: expected value: "%d", got: "%d", for test: "%s"`, test_int, got, test_name)
		}
	})

	test_name = "BigInt - math/big"
	t.Run(test_name, func(t *testing.T) {
		test_bigint_str := "123456789012345678901234567890"
		test_bigint, _ := (&big.Int{}).SetString(test_bigint_str, 10)
		val := ctx.NewBigInt(test_bigint)
		defer val.Free()
		if !val.IsBigInt() {
			t.Errorf(`[type check ]: expected js-type "bigint" for test: "%s"`, test_name)
		}
		if got := val.ToBigInt(); got.Cmp(test_bigint) != 0 {
			t.Errorf(`[value check]: expected value: "%s", got: "%s", for test: "%s"`, test_bigint.String(), got.String(), test_name)
		}
	})
}

func TestValue_Strings(t *testing.T) {
	rt := js.NewRuntime()
	defer rt.Free()
	ctx := rt.NewContext()
	defer ctx.Free()

	test_name := "String"
	test_str := "hello world!"
	t.Run(test_name, func(t *testing.T) {
		val := ctx.NewString(test_str)
		defer val.Free()
		if !val.IsString() {
			t.Errorf(`[type check ]: expected js-type "string" for test: "%s"`, test_name)
		}
		if got := val.ToString(); got != test_str {
			t.Errorf(`[value check]: expected value: "%s", got: "%s", for test: "%s"`, test_str, got, test_name)
		}
	})

	test_name = "String - with null character"
	test_str = "hello \x00 world!"
	t.Run(test_name, func(t *testing.T) {
		val := ctx.NewString(test_str)
		defer val.Free()
		if !val.IsString() {
			t.Errorf(`[type check ]: expected js-type "string" for test: "%s"`, test_name)
		}
		if got := val.ToString(); got != test_str {
			t.Errorf(`[value check]: expected value: "%s", got: "%s", for test: "%s"`, test_str, got, test_name)
		}
	})
}

func TestValue_Symbols(t *testing.T) {
	rt := js.NewRuntime()
	defer rt.Free()
	ctx := rt.NewContext()
	defer ctx.Free()

	test_name := "Symbol"
	test_description := "some unique symbol!"
	t.Run(test_name, func(t *testing.T) {
		js_str := ctx.NewString(test_description)
		defer js_str.Free()
		sym := ctx.NewSymbol(js_str)
		defer sym.Free()
		if !sym.IsSymbol() {
			t.Errorf(`[type check ]: expected js-type "symbol" for test: "%s"`, test_name)
		}
		sym_description := sym.Get("description")
		defer sym_description.Free()
		if got := sym_description.ToString(); got != test_description {
			t.Errorf(`[value check]: expected description of symbol to be: "%s", got: "%s", for test: "%s"`, test_description, got, test_name)
		}
	})

	test_name = "Symbol - integer description"
	test_description_int := int32(42)
	t.Run(test_name, func(t *testing.T) {
		js_str := ctx.NewInt32(test_description_int)
		sym := ctx.NewSymbol(js_str)
		defer sym.Free()
		if !sym.IsSymbol() {
			t.Errorf(`[type check ]: expected js-type "symbol" for test: "%s"`, test_name)
		}
		sym_description := sym.Get("description") // even though the original description was a number, it becomes a string once assigned to the symbol.
		defer sym_description.Free()
		if got := sym_description.ToInt32(); got != test_description_int {
			t.Errorf(`[value check]: expected description of symbol to be: "%d", got: "%d", for test: "%s"`, test_description_int, got, test_name)
		}
	})
}

func TestValue_GlobalThis(t *testing.T) {
	rt := js.NewRuntime()
	defer rt.Free()
	ctx := rt.NewContext()
	defer ctx.Free()
	global_this := ctx.GetGlobalThis() // should not be freed!
	if !global_this.IsObject() {
		t.Errorf(`[type check ]: expected "globalThis" to be an "object"`)
	}
}
