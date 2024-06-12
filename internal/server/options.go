package server

type ServerOption func(*server)

func WithCertPath(certPath string) ServerOption {
	return func(s *server) {
		s.tlsCertPath = certPath
	}
}

func WithKeyPath(keyPath string) ServerOption {
	return func(s *server) {
		s.tlsKeyPath = keyPath
	}
}

func WithPort(port int) ServerOption {
	return func(s *server) {
		s.port = port
	}
}
