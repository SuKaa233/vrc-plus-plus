package security

type Protector interface {
	Protect(plain []byte) ([]byte, error)
	Unprotect(cipher []byte) ([]byte, error)
	Name() string
}
