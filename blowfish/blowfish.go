//go:build !solution

package blowfish

/* #cgo pkg-config: libcrypto
#cgo CFLAGS: -Wno-deprecated-declarations
#include <openssl/blowfish.h>
typedef const unsigned char* const_ptr;
typedef void* void_ptr;
*/
import "C"
import (
	"unsafe"
)

type Blowfish struct {
	key C.BF_KEY
}


func New(key []byte) *Blowfish {
	obj := &Blowfish{}
	C.BF_set_key((*C.BF_KEY)(unsafe.Pointer(&obj.key)), (C.int)(len(key)), (C.const_ptr)((C.void_ptr)(unsafe.SliceData(key))))
	return obj
}


func (b *Blowfish) Encrypt(dst, src []byte) {
	C.BF_ecb_encrypt((C.const_ptr)((C.void_ptr)(unsafe.SliceData(src))),
		(C.const_ptr)((C.void_ptr)(unsafe.SliceData(dst))), &b.key, C.BF_ENCRYPT)
}


func (b *Blowfish) Decrypt(dst, src []byte) {
	C.BF_ecb_encrypt((C.const_ptr)((C.void_ptr)(unsafe.SliceData(src))),
		(C.const_ptr)((C.void_ptr)(unsafe.SliceData(dst))), &b.key, C.BF_DECRYPT)
}

func (b *Blowfish) BlockSize() int {
	return 8
}
