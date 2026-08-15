package security

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	crypt32            = windows.NewLazySystemDLL("crypt32.dll")
	cryptProtectData   = crypt32.NewProc("CryptProtectData")
	cryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
	kernel32           = windows.NewLazySystemDLL("kernel32.dll")
	localFree          = kernel32.NewProc("LocalFree")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

type DPAPIProtector struct{}

func NewProtector() Protector { return DPAPIProtector{} }

func (DPAPIProtector) Name() string { return "Windows DPAPI (CurrentUser)" }

func (DPAPIProtector) Protect(plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return []byte{}, nil
	}
	in := bytesToBlob(plain)
	var out dataBlob
	result, _, callErr := cryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
	)
	if result == 0 {
		return nil, fmt.Errorf("CryptProtectData: %w", callErr)
	}
	defer localFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return blobToBytes(out), nil
}

func (DPAPIProtector) Unprotect(cipher []byte) ([]byte, error) {
	if len(cipher) == 0 {
		return []byte{}, nil
	}
	in := bytesToBlob(cipher)
	var out dataBlob
	result, _, callErr := cryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
	)
	if result == 0 {
		return nil, fmt.Errorf("CryptUnprotectData: %w", callErr)
	}
	defer localFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return blobToBytes(out), nil
}

func bytesToBlob(value []byte) dataBlob {
	return dataBlob{cbData: uint32(len(value)), pbData: &value[0]}
}

func blobToBytes(blob dataBlob) []byte {
	if blob.cbData == 0 || blob.pbData == nil {
		return []byte{}
	}
	return append([]byte(nil), unsafe.Slice(blob.pbData, blob.cbData)...)
}
