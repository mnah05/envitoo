## Goal Description
Integrate your Envitoo platform with **Grafana Cloud** (the managed cloud version of Prometheus, Loki, and Grafana) instead of running the entire heavy stack locally. This is a great choice as it saves local CPU/RAM and mirrors a production setup where telemetry is sent off-site.

We will run a lightweight data collector locally (Grafana Alloy) to scrape your metrics and logs and securely push them to your Grafana Cloud account.

## User Review Required
> [!IMPORTANT]
> **Grafana Cloud Account Setup**
> You will need to create a free Grafana Cloud account if you haven't already. 
> To complete this implementation, you will need to generate and provide:
> 1. Your Prometheus **Remote Write Endpoint URL**, **Username**, and **Access Policy Token**.
> 2. Your Loki **Push URL**, **Username**, and **Access Policy Token**.

## Open Questions
*(All open questions have been resolved in the chat).*

## Proposed Changes

---

### Observability Infrastructure (Docker Compose)
We will add a single lightweight collector to your `docker-compose.yml` to push telemetry to the cloud.

#### [NEW] `docker/alloy/config.alloy`
The configuration file for Grafana Alloy to scrape Docker logs and your Go `/metrics`, then remote-write them to Grafana Cloud.

```hcl
// Scrape Go Backend Metrics
prometheus.scrape "backend" {
  targets = [
    {"__address__" = "backend:8080"},
  ]
  forward_to = [prometheus.remote_write.grafanacloud.receiver]
  scrape_interval = "15s"
}

// Push Metrics to Grafana Cloud
prometheus.remote_write "grafanacloud" {
  endpoint {
    url = sys.env("GRAFANA_CLOUD_PROMETHEUS_URL")
    basic_auth {
      username = sys.env("GRAFANA_CLOUD_PROMETHEUS_USER")
      password = sys.env("GRAFANA_CLOUD_API_KEY")
    }
  }
}

// Scrape Docker Logs (simplified)
loki.source.docker "local_containers" {
  host       = "unix:///var/run/docker.sock"
  targets    = discovery.docker.local_containers.targets
  forward_to = [loki.write.grafanacloud.receiver]
}

discovery.docker "local_containers" {
  host = "unix:///var/run/docker.sock"
}

// Push Logs to Grafana Cloud
loki.write "grafanacloud" {
  endpoint {
    url = sys.env("GRAFANA_CLOUD_LOKI_URL")
    basic_auth {
      username = sys.env("GRAFANA_CLOUD_LOKI_USER")
      password = sys.env("GRAFANA_CLOUD_API_KEY")
    }
  }
}
```

#### [MODIFY] `docker-compose.yml`
Add the lightweight Alloy collector and pass the Grafana Cloud credentials as environment variables.

```yaml
  alloy:
    image: grafana/alloy:latest
    volumes:
      - ./docker/alloy/config.alloy:/etc/alloy/config.alloy:ro
      - /var/run/docker.sock:/var/run/docker.sock:ro
    command: run --server.http.listen-addr=0.0.0.0:12345 /etc/alloy/config.alloy
    env_file:
      - .env.grafana-cloud # You will create this file with your keys
    profiles:
      - full
      - observability
```

---

### Backend (Go / Echo)

#### [MODIFY] `backend/go.mod`
Add the Echo Prometheus middleware:
```bash
go get github.com/labstack/echo-contrib/echoprometheus
```

#### [MODIFY] `backend/internal/server/server.go`
Add the Prometheus middleware and the `/metrics` endpoint to the Echo router. 

*Note on Latency:* The `echoprometheus` middleware **automatically** tracks request/response latency via a histogram metric named `envitoo_request_duration_seconds`. However, if you want to track a custom business-logic latency (e.g., how long a specific DB query takes), we can also register a custom Prometheus Histogram.

```go
import (
    "github.com/labstack/echo-contrib/echoprometheus"
    "github.com/prometheus/client_golang/prometheus"
)

// Example of a custom metric (optional, if you want to track things beyond HTTP requests)
var customLatencyMetric = prometheus.NewHistogram(
    prometheus.HistogramOpts{
        Name:    "envitoo_custom_operation_duration_seconds",
        Help:    "Latency of a specific custom operation.",
        Buckets: prometheus.DefBuckets,
    },
)

func init() {
    prometheus.MustRegister(customLatencyMetric)
}

func (s *Server) Register(ctx context.Context, e *echo.Echo) {
    // ... existing global middleware ...
    
    // Add Prometheus middleware to measure all requests (This automatically gives you HTTP request latency!)
    e.Use(echoprometheus.NewMiddleware("envitoo"))

    e.GET("/healthz", s.Health)
    
    // Add the /metrics endpoint for Alloy to scrape
    e.GET("/metrics", echoprometheus.NewHandler())

    // ... rest of routes ...
}
```

## Verification Plan

### Manual Verification
1. Create a `.env.grafana-cloud` file with your Grafana Cloud URLs and API keys.
2. Start infrastructure: `make dc-full`.
3. Generate some traffic by navigating your app and hitting API endpoints.
4. Log into your **Grafana Cloud Dashboard** in your browser.
5. Open **Explore**, select your cloud Loki instance, and query `{container=~".+"}` to verify logs are arriving.
6. Still in **Explore**, select your cloud Prometheus instance, and query `http_requests_total` to verify metrics are arriving.
