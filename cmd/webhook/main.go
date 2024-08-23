package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"

	"k8s-admission-webhook/pkg/webhook"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

func init() {
	_ = scheme.AddToScheme(runtimeScheme)
}

func main() {
	port := 8443
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		if envPort, err := strconv.Atoi(portEnv); err == nil {
			port = envPort
		}
	}

	// Create the webhook server with handlers
	mux := http.NewServeMux()
	whServer := &webhook.WebhookServer{}

	mux.HandleFunc("/validate", whServer.ValidateHandler)
	mux.HandleFunc("/mutate", whServer.MutateHandler)
	mux.HandleFunc("/healthz", whServer.HealthHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
		TLSConfig: &tls.Config{
			ClientAuth: tls.NoClientCert,
		},
	}

	// Load TLS certificates
	_, err := tls.LoadX509KeyPair("/certs/tls.crt", "/certs/tls.key")
	if err != nil {
		log.Printf("Failed to load TLS certificates: %v", err)
		log.Println("Using self-signed certificates for development")

		// Generate self-signed certificates for development
		_, err = generateSelfSignedCert()
		if err != nil {
			log.Fatalf("Failed to generate self-signed certificate: %v", err)
		}
	}

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting webhook server on port %d", port)
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

// Helper function to generate self-signed certificates for development
func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(24 * time.Hour)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"k8s-webhook"},
		},
		DNSNames:              []string{"localhost", "webhook-service", "webhook-service.default", "webhook-service.default.svc"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return tls.X509KeyPair(certPEM, privPEM)
}// Webhook server implementation
// Webhook server
// Webhook server implementation
