#!/bin/bash
#!/bin/bash
#!/bin/bash

set -e

# Configuration
SERVICE_NAME="admission-webhook"
NAMESPACE="webhook-system"
SECRET_NAME="webhook-certs"

# Generate CA and server certificates
echo "Generating CA certificate and key..."
openssl genrsa -out ca.key 2048
openssl req -new -x509 -key ca.key -out ca.crt -days 365 -subj "/CN=webhook-ca"

echo "Generating server certificate and key..."
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -subj "/CN=${SERVICE_NAME}.${NAMESPACE}.svc" -config <(
  cat <<EOF
[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = v3_req

[dn]
CN=${SERVICE_NAME}.${NAMESPACE}.svc

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${SERVICE_NAME}
DNS.2 = ${SERVICE_NAME}.${NAMESPACE}
DNS.3 = ${SERVICE_NAME}.${NAMESPACE}.svc
DNS.4 = ${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local
IP.1 = 127.0.0.1
EOF
)

echo "Signing server certificate with CA..."
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -extensions v3_req -extfile <(
  cat <<EOF
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${SERVICE_NAME}
DNS.2 = ${SERVICE_NAME}.${NAMESPACE}
DNS.3 = ${SERVICE_NAME}.${NAMESPACE}.svc
DNS.4 = ${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local
IP.1 = 127.0.0.1
EOF
)

# Create Kubernetes secret
echo "Creating Kubernetes secret..."
kubectl create secret tls ${SECRET_NAME} \
  --cert=server.crt \
  --key=server.key \
  --namespace=${NAMESPACE} \
  --dry-run=client -o yaml >webhook-certs-secret.yaml

# Update the webhook configurations with the CA bundle
CA_BUNDLE=$(cat ca.crt | base64 | tr -d '\n')
sed -i.bak "s|caBundle: \"\"|caBundle: \"${CA_BUNDLE}\"|g" validating-webhook-config.yaml
sed -i.bak "s|caBundle: \"\"|caBundle: \"${CA_BUNDLE}\"|g" mutating-webhook-config.yaml

# Cleanup
rm -f ca.key ca.crt ca.srl server.csr server.key server.crt

echo "Certificate generation complete!"
echo "Secret saved to webhook-certs-secret.yaml"
echo "Webhook configurations updated with CA bundle"

