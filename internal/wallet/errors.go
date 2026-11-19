package wallet

import "errors"

var (
	// ErrWalletNotInitialized is returned when a wallet operation runs before InitWallet or SetWallet.
	ErrWalletNotInitialized = errors.New("wallet not initialized")

	// ErrMissingServerPrivateKey is returned by InitWallet when no server private key is configured.
	ErrMissingServerPrivateKey = errors.New("SERVER_PRIVATE_KEY is required for wallet initialization")

	// ErrNoMatchedOutput is returned when a renewal has no existing advertisement output to spend.
	ErrNoMatchedOutput = errors.New("no matched output found")

	// ErrTransactionNotInBEEF is returned when the signable renewal transaction is missing from its BEEF.
	ErrTransactionNotInBEEF = errors.New("transaction not found in BEEF")

	// ErrInputNotFound is returned when the signable renewal transaction does not spend the matched output.
	ErrInputNotFound = errors.New("could not find input matching outpoint in signable transaction")

	// ErrInvalidPushDropScript is returned when an advertisement output is not a decodable PushDrop script.
	ErrInvalidPushDropScript = errors.New("invalid or missing pushdrop locking script")

	// ErrNegativeValue is returned when an advertisement expiry or content length is negative.
	ErrNegativeValue = errors.New("value must not be negative")

	// ErrInvalidHMACFormat is returned when an upload HMAC is not 32 hex-encoded bytes.
	ErrInvalidHMACFormat = errors.New("invalid HMAC format")

	// ErrInvalidHMAC is returned when an upload HMAC does not verify.
	ErrInvalidHMAC = errors.New("invalid HMAC")

	// ErrAdvertisementNotFound is returned when no advertisement matches a UHRP URL and uploader.
	ErrAdvertisementNotFound = errors.New("uhrpUrl not found in wallet outputs")

	// ErrMultipleAdvertisements is returned when more than one advertisement matches a UHRP URL and uploader.
	ErrMultipleAdvertisements = errors.New("multiple advertisements found for uhrpUrl")

	// ErrAdvertisementExpired is returned when the matching advertisement has already expired.
	ErrAdvertisementExpired = errors.New("advertisement for uhrpUrl is expired")

	// ErrInvalidFileSize is returned when an advertisement's size tag is missing or not an integer.
	ErrInvalidFileSize = errors.New("invalid file size in advertisement")
)
