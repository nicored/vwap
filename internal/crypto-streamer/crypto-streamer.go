package crypto_streamer

type Streamer interface {
	Subscribe(channels []string, productIDs []string) error
	Unsubscribe() error
}
