// this file contains the cgo/gcc compilation flags for the c-source code of quickjs.
// there are two modes available: 1) the default debug mode, 2) the release mode.
// 1) in the debug mode, your binaries will use the precompiled static quickjs library ("libquickjs.a") for quick builds.
// 2) in the release mode, activated using `go build -tags="quickcc_release" ...`,
//    quickjs will be built from the c-source, leading to a better crossplatform compatibility.

package bridge

/*
#cgo CFLAGS: -I"${SRCDIR}/"

// only use the "-O2" optimization flag when building in release mode.
#cgo quickcc_release CFLAGS: -O2

// only link to the pre-compiled static library when not in release mode.
#cgo (!quickcc_release && linux   && amd64) LDFLAGS: -L"${SRCDIR}/../../vendor/quickjs_lib/linux_amd64/"   -lquickjs
#cgo (!quickcc_release && windows && amd64) LDFLAGS: -L"${SRCDIR}/../../vendor/quickjs_lib/windows_amd64/" -lquickjs
#cgo (!quickcc_release && darwin  && amd64) LDFLAGS: -L"${SRCDIR}/../../vendor/quickjs_lib/darwin_amd64/"  -lquickjs
#cgo (!quickcc_release && freebsd && amd64) LDFLAGS: -L"${SRCDIR}/../../vendor/quickjs_lib/freebsd_amd64/" -lquickjs

// compile-time safety checks.
#cgo !windows CFLAGS: -Wall -Wno-array-bounds -Wno-format-truncation -Wno-infinite-recursion
#cgo windows CFLAGS: -Wall

// including the math library.
#cgo LDFLAGS: -lm
// include posix threads if targeting windows.
// (I think quickjs doesn't actually require pthreads on windows. it is only needed when targeting windows on a linux host.)
#cgo windows LDFLAGS: -lpthread

// in each file that shall reference quickjs c-constructs, we must use the following include statement to include its header file:
// #include "./include0_quickjs.h"
*/
import "C"
