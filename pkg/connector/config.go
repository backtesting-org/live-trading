package connector

// Config is the base interface all exchange configs must implement
type Config interface {
	// Validate ensures the configuration is valid
	Validate() error

	// ExchangeName returns the name of the exchange this config is for
	ExchangeName() string
}
