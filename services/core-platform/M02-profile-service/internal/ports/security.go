package ports

type Encryption interface {
	Encrypt(userID string, value string) ([]byte, error)
	Decrypt(userID string, payload []byte) (string, error)
}
