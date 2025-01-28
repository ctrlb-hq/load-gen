# Log Generator

## Overview

The Log Generator is a Go-based application designed to generate and send logs at a configurable rate. These logs are batched and sent to a specified HTTP endpoint. This tool is ideal for load testing, log simulation, and validating observability pipelines.

---

## Features

- **Configurable Log Rate:** Control the rate of log generation via environment variables.
- **Batching:** Logs are grouped into batches for efficient processing and delivery.
- **Custom Endpoints:** Logs can be sent to any HTTP endpoint specified via configuration.
- **Authentication Support:** Supports passing an authentication header.
- **Randomized Log Content:** Uses `gofakeit` to generate realistic log data.

---

## Environment Variables

The application behavior can be customized using the following environment variables:

| Variable       | Description                                    | Default Value   |
| -------------- | ---------------------------------------------- | --------------- |
| `LOG_RATE`     | Number of logs generated per second.           | `1`             |
| `BATCH_SIZE`   | Number of logs in a single batch.              | `1000`          |
| `LOG_ENDPOINT` | The HTTP endpoint to which logs are sent.      | None (required) |
| `AUTH_HEADER`  | Authorization header for secure communication. | None            |

---

## Usage

### Prerequisites

- Go 1.23.4 or higher installed.
- [gofakeit](https://github.com/brianvoe/gofakeit) for generating fake log data.

### Run Locally

1. Clone the repository:

   ```bash
   git clone https://github.com/your-repo/log-generator.git
   cd log-generator
   ```

2. Set up environment variables:

   ```bash
   export LOG_RATE=5
   export BATCH_SIZE=500
   export LOG_ENDPOINT=https://example.com/api/logs
   export AUTH_HEADER="<your-auth-header>"
   ```

3. Build and run the application:

   ```bash
   go build -o log-generator .
   ./log-generator
   ```

### Using Docker

1. Build the Docker image:

   ```bash
   docker build -t log-generator:latest .
   ```

2. Run the container:

   ```bash
   docker run -e LOG_RATE=5 -e BATCH_SIZE=500 -e LOG_ENDPOINT=https://example.com/api/logs -e AUTH_HEADER="<your-auth-header>" log-generator:latest
   ```

---

## Kubernetes Deployment

For deploying the application on Kubernetes, use the provided `k8s_deployment.yaml` file:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: log-generator
  labels:
    app: log-generator
spec:
  replicas: 2
  selector:
    matchLabels:
      app: log-generator
  template:
    metadata:
      labels:
        app: log-generator
    spec:
      containers:
      - name: log-generator
        image: ghcr.io/your-org/log-generator:latest
        env:
        - name: LOG_ENDPOINT
          value: "https://example.com/api/logs"
        - name: AUTH_HEADER
          value: "<your-auth-header>"
        ports:
        - containerPort: 8080
```

Deploy it using:

```bash
kubectl apply -f k8s_deployment.yaml
```

---

## Development

### Run Tests

Run the unit tests using:

```bash
go test ./...
```

### Contribute

1. Fork the repository.
2. Create a new feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. Commit your changes and create a pull request.

---

## License

This project is licensed under the Apache 2.0 License. See the LICENSE file for more details.

