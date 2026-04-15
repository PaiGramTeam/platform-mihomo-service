package main

import (
	"testing"

	"platform-mihomo-service/internal/conf"
)

func TestValidateBootstrap(t *testing.T) {
	t.Run("accepts valid bootstrap", func(t *testing.T) {
		bc := &conf.Bootstrap{
			Server: &conf.Server{
				Grpc: &conf.Server_GRPC{
					Network:        "tcp",
					Addr:           "127.0.0.1:9000",
					TimeoutSeconds: 5,
				},
			},
			Security: &conf.Security{
				CredentialEncryptionKey: "0123456789abcdef0123456789abcdef",
				ServiceTicketSigningKey: "fedcba9876543210fedcba9876543210",
				ServiceTicketIssuer:     "paigram-account-center",
			},
		}

		if err := validateBootstrap(bc); err != nil {
			t.Fatalf("validateBootstrap() error = %v", err)
		}
	})

	t.Run("rejects missing grpc address", func(t *testing.T) {
		bc := &conf.Bootstrap{
			Server: &conf.Server{
				Grpc: &conf.Server_GRPC{
					Network:        "tcp",
					TimeoutSeconds: 5,
				},
			},
			Security: &conf.Security{
				CredentialEncryptionKey: "0123456789abcdef0123456789abcdef",
				ServiceTicketSigningKey: "fedcba9876543210fedcba9876543210",
				ServiceTicketIssuer:     "paigram-account-center",
			},
		}

		if err := validateBootstrap(bc); err == nil {
			t.Fatal("validateBootstrap() error = nil, want non-nil")
		}
	})

	t.Run("rejects short signing key", func(t *testing.T) {
		bc := &conf.Bootstrap{
			Server: &conf.Server{
				Grpc: &conf.Server_GRPC{
					Network:        "tcp",
					Addr:           "127.0.0.1:9000",
					TimeoutSeconds: 5,
				},
			},
			Security: &conf.Security{
				CredentialEncryptionKey: "0123456789abcdef0123456789abcdef",
				ServiceTicketSigningKey: "short-key",
				ServiceTicketIssuer:     "paigram-account-center",
			},
		}

		if err := validateBootstrap(bc); err == nil {
			t.Fatal("validateBootstrap() error = nil, want non-nil")
		}
	})
}
