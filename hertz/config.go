package main

type Config struct {
	Cache          Cache          `json:"cache"`
	Auth           Auth           `json:"auth"`
	Retry          Retry          `json:"retry"`
	CircuitBreaker CircuitBreaker `json:"circuitBreaker"`
}

type Cache struct {
	EnableCache bool `json:"enableCache"`
	WithHeader  bool `json:"withHeader"`
	WithBody    bool `json:"withBody"`
	WithURI     bool `json:"withURI"`
}

type Auth struct {
	EnableAuth bool   `json:"enableAuth"`
	Token      string `json:"token"`
}

type Retry struct {
	EnableRetry bool `json:"enableRetry"`
	MaxTimes    int  `json:"maxTimes"`
}

type CircuitBreaker struct {
	EnableCircuitBreaker bool `json:"enableCircuitBreaker"`
}
