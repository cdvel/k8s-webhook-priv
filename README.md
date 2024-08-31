# Kubernetes Admission Webhook

A production-grade Kubernetes validating and mutating admission webhook that enforces security policies and best practices for pod deployments.

## Overview

This webhook intercepts pod creation and update requests to Kubernetes API server and applies security validations and configurations automatically. It ensures all pods meet organizational security standards before being admitted to the cluster.

## Features

- **Security Validation**: Enforces non-root execution, resource limits, and approved image registries
- **Automatic Mutations**: Injects standard labels, annotations, and security contexts
- **Resource Governance**: Ensures proper CPU and memory limits/requests
- **TLS Security**: Full HTTPS communication with proper certificates
- **Observability**: Structured logging and metrics

## Architecture

The webhook implements both validating and mutating admission controllers:

- **Validating Webhook**: Rejects non-compliant pods based on security policies
- **Mutating Webhook**: Automatically injects required configurations into pods

## Quick Start

1. Build the Docker image:
   ```bash
   make docker-build
   ```

2. Generate certificates:
   ```bash
   make certs
   ```

3. Deploy to Kubernetes:
   ```bash
   make deploy
   ```

## Security Policies

The webhook enforces the following security policies:

- All pods must run as non-root user
- All containers must have resource limits and requests
- Only approved image registries are allowed
- Privileged containers are prohibited
- Read-only root filesystem required

## Examples

See example pod definitions in the `examples/` directory:

- `compliant-pod.yaml` - A pod that meets all security requirements
- `noncompliant-pod.yaml` - A pod that will be rejected by the webhook

## Development

Run tests:
```bash
make test
```

Run with coverage:
```bash
make test-coverage
```

Format code:
```bash
make fmt
```

## Deployment

The webhook is designed for production deployment with:

- Proper RBAC configuration
- TLS certificate management
- Health checks and readiness probes
- Resource limits and requests

## License

MIT
# README
# README
