<p align="center">
	<a href="https://caddyserver.com">
		<picture>
			<source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/1128849/210187358-e2c39003-9a5e-4dd5-a783-6deb6483ee72.svg">
			<source media="(prefers-color-scheme: light)" srcset="https://user-images.githubusercontent.com/1128849/210187356-dfb7f1c5-ac2e-43aa-bb23-fc014280ae1f.svg">
			<img src="https://user-images.githubusercontent.com/1128849/210187356-dfb7f1c5-ac2e-43aa-bb23-fc014280ae1f.svg" alt="Caddy" width="550">
		</picture>
	</a>
	<br>
	<h3 align="center">a <a href="https://zerossl.com"><img src="https://user-images.githubusercontent.com/55066419/208327323-2770dc16-ec09-43a0-9035-c5b872c2ad7f.svg" height="28" style="vertical-align: -7.7px" valign="middle"></a> project</h3>
</p>
<hr>
<h3 align="center">Every site on HTTPS</h3>
<p align="center">Caddy is an extensible server platform that uses TLS by default.</p>
<p align="center">
	<a href="https://github.com/caddyserver/caddy/releases">Releases</a> ·
	<a href="https://caddyserver.com/docs/">Documentation</a> ·
	<a href="https://caddy.community">Get Help</a>
</p>
<p align="center">
	<a href="https://github.com/caddyserver/caddy/actions/workflows/ci.yml"><img src="https://github.com/caddyserver/caddy/actions/workflows/ci.yml/badge.svg"></a>
	&nbsp;
	<a href="https://www.bestpractices.dev/projects/7141"><img src="https://www.bestpractices.dev/projects/7141/badge"></a>
	&nbsp;
	<a href="https://pkg.go.dev/github.com/caddyserver/caddy/v2"><img src="https://img.shields.io/badge/godoc-reference-%23007d9c.svg"></a>
	&nbsp;
	<a href="https://x.com/caddyserver" title="@caddyserver on Twitter"><img src="https://img.shields.io/twitter/follow/caddyserver" alt="@caddyserver on Twitter"></a>
	&nbsp;
	<a href="https://caddy.community" title="Caddy Forum"><img src="https://img.shields.io/badge/community-forum-ff69b4.svg" alt="Caddy Forum"></a>
	<br>
	<a href="https://sourcegraph.com/github.com/caddyserver/caddy?badge" title="Caddy on Sourcegraph"><img src="https://sourcegraph.com/github.com/caddyserver/caddy/-/badge.svg" alt="Caddy on Sourcegraph"></a>
	&nbsp;
	<a href="https://cloudsmith.io/~caddy/repos/"><img src="https://img.shields.io/badge/OSS%20hosting%20by-cloudsmith-blue?logo=cloudsmith" alt="Cloudsmith"></a>
</p>
<p align="center">
	<b>Powered by</b>
	<br>
	<a href="https://github.com/caddyserver/certmagic">
		<picture>
			<source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/55066419/206946718-740b6371-3df3-4d72-a822-47e4c48af999.png">
			<source media="(prefers-color-scheme: light)" srcset="https://user-images.githubusercontent.com/1128849/49704830-49d37200-fbd5-11e8-8385-767e0cd033c3.png">
			<img src="https://user-images.githubusercontent.com/1128849/49704830-49d37200-fbd5-11e8-8385-767e0cd033c3.png" alt="CertMagic" width="250">
		</picture>
	</a>
</p>

<!-- Warp sponsorship requests this section -->
<div align="center" markdown="1">
	<hr>
	<sup>Special thanks to:</sup>
	<br>
	<a href="https://go.warp.dev/caddy">
		<img alt="Warp sponsorship" width="400" src="https://github.com/user-attachments/assets/c8efffde-18c7-4af4-83ed-b1aba2dda394">
	</a>

### [Warp, built for coding with multiple AI agents](https://go.warp.dev/caddy)
[Available for MacOS, Linux, & Windows](https://go.warp.dev/caddy)<br>
</div>

<hr>

### Menu

- [Features](#features)
- [Install](#install)
- [Build from source](#build-from-source)
	- [For development](#for-development)
	- [With version information and/or plugins](#with-version-information-andor-plugins)
- [Quick start](#quick-start)
- [Overview](#overview)
- [Full documentation](#full-documentation)
- [Getting help](#getting-help)
- [About](#about)


## ProxFlow Enhancements

ProxFlow is a cloud-native reverse proxy built on Caddy, featuring advanced load balancing capabilities.

### AI-Powered Load Balancing

ProxFlow introduces an AI-powered load balancing feature that uses machine learning to predict traffic patterns and dynamically adjust load balancing strategies.

#### How it Works

The `adaptive` selection policy collects historical request counts for each upstream server and uses exponential smoothing to predict future load. It then assigns weights inversely proportional to the predicted load, directing more traffic to less busy servers.

#### Configuration

In your Caddyfile, use:

```
lb_policy adaptive {
    history_length 10
    update_interval 1m
    alpha 0.3
}
```

- `history_length`: Number of past intervals to keep for prediction (default 10)
- `update_interval`: How often to update predictions (default 1m)
- `alpha`: Smoothing factor for exponential smoothing (0-1, default 0.3)

#### 10-Turn Zed Eval Prompt for AI-Powered Load Balancing

This prompt is designed to evaluate the AI-powered load balancing feature through a 10-turn conversation. It can be used with conversational AIs to assess understanding and implementation.

**Turn 1:** What is the new AI-powered load balancing feature in ProxFlow?

**Turn 2:** How does the adaptive selection policy work?

**Turn 3:** What machine learning technique is used for traffic prediction?

**Turn 4:** How are the weights adjusted based on predictions?

**Turn 5:** What are the configuration options for the adaptive policy?

**Turn 6:** How often does the policy update its predictions?

**Turn 7:** What happens if an upstream has no historical data?

**Turn 8:** How does the policy handle unavailable upstreams?

**Turn 9:** What are the default values for the adaptive policy parameters?

**Turn 10:** How can I monitor the effectiveness of the AI-powered load balancing?

## GraphQL Federation Gateway

ProxFlow includes a GraphQL Federation Gateway that can federate multiple GraphQL services into a unified API endpoint. This is particularly useful in microservices architectures where each service exposes its own GraphQL API.

### Features

- **Schema Stitching**: Automatically merges schemas from multiple GraphQL services.
- **Intelligent Query Planning**: Plans queries across services for efficient execution.
- **Caching**: Caches query plans and responses to improve performance.
- **Complexity Limiting**: Prevents abuse by limiting query complexity.
- **Introspection Support**: Optional support for GraphQL introspection.

### Configuration

In your Caddyfile:

```
graphql_federation {
    service user_service http://user-service:4001/graphql {
        schema `
            type User {
                id: ID!
                name: String!
            }
            type Query {
                user(id: ID!): User
            }
        `
    }
    service product_service http://product-service:4002/graphql {
        schema `
            type Product {
                id: ID!
                name: String!
            }
            type Query {
                product(id: ID!): Product
            }
        `
    }
    cache_duration 5m
    max_query_complexity 1000
    enable_introspection
}
```

- `service`: Defines an upstream GraphQL service with name, URL, and schema.
- `cache_duration`: How long to cache plans and responses (default 5m).
- `max_query_complexity`: Maximum allowed query complexity (default 1000).
- `enable_introspection`: Allows GraphQL introspection queries.

### Usage

The gateway exposes a single GraphQL endpoint that can query across all federated services. For example, if you have user and product services, you can query both in a single request.

## Real-Time Traffic Visualization Dashboard

ProxFlow includes a web-based dashboard for real-time traffic monitoring and visualization.

### Features

- **Real-Time Metrics**: Request rates, error rates, and active connections
- **Latency Monitoring**: P50, P95, P99 latency percentiles
- **Geographic Distribution**: Requests by region
- **Live Updates**: Server-Sent Events (SSE) for real-time data streaming
- **Interactive Dashboard**: Web interface for monitoring traffic patterns

### Configuration

In your Caddyfile:

```
traffic_dashboard {
    path /dashboard
    update_interval 1s
}
```

- `path`: The URL path where the dashboard is served (default `/dashboard`)
- `update_interval`: How often to send updates (default 1s)

### Usage

Navigate to the configured path (e.g., `http://your-server/dashboard`) to view the dashboard. The page will automatically update with real-time traffic data via Server-Sent Events.

## Intelligent Circuit Breaker with Auto Recovery

ProxFlow includes an intelligent circuit breaker that monitors backend health using multiple metrics and provides automatic recovery with traffic ramp-up.

### Features

- **Multi-Metric Monitoring**: Tracks latency, error rates, and timeouts
- **Automatic Circuit Management**: Opens/closes circuits based on configurable thresholds
- **Smart Recovery**: Half-open state for testing recovery, followed by ramp-up phase
- **Traffic Ramp-Up**: Gradual traffic increase during recovery to prevent backend overload
- **Configurable Parameters**: Thresholds, timeouts, and ramp-up durations

### Configuration

In your Caddyfile, configure per upstream:

```
reverse_proxy localhost:8080 {
    circuit_breaker intelligent {
        failure_threshold 0.5
        recovery_timeout 60s
        half_open_max_requests 3
        latency_threshold 5s
        min_requests 10
        evaluation_window 60s
        ramp_up_duration 30s
    }
}
```

### Parameters

- `failure_threshold`: Failure rate (0-1) to trigger circuit open (default 0.5)
- `recovery_timeout`: Time to wait before attempting recovery (default 60s)
- `half_open_max_requests`: Max requests in half-open state (default 3)
- `latency_threshold`: Latency threshold for slow requests (default 5s)
- `min_requests`: Minimum requests needed for evaluation (default 10)
- `evaluation_window`: Time window for metric evaluation (default 60s)
- `ramp_up_duration`: Duration of traffic ramp-up phase (default 30s)

### Circuit States

1. **Closed**: Normal operation, all requests pass through
2. **Open**: Circuit is open, requests are blocked
3. **Half-Open**: Testing recovery with limited requests
4. **Ramp-Up**: Gradual traffic increase during recovery

## API Gateway with Request Transformation

ProxFlow includes an API gateway with powerful request/response transformation capabilities.

### Features

- **Request/Response Transformation**: Modify headers, body, and structure
- **Request Enrichment**: Add additional data from external sources
- **Response Aggregation**: Combine responses from multiple backends
- **Flexible Matching**: Apply transformations based on request conditions

### Configuration

In your Caddyfile:

```
transform {
	header add X-API-Version "v1"
	body replace user.name "enriched"
	enrich {
		data key1 value1
		from_url https://api.example.com/enrich
	}
	aggregate {
		backend https://backend1.com/api
		backend https://backend2.com/api
		combine merge
	}
}
```

### Transformation Types

- **Header Actions**: add, set, delete headers
- **Body Actions**: replace, append, prepend JSON values
- **Enrichment**: Add data from external APIs
- **Aggregation**: Combine multiple backend responses

## Multi-Tenancy with Dynamic Routing

ProxFlow supports multi-tenant architectures with isolated configurations and dynamic tenant management.

### Features

- **Tenant Isolation**: Separate routing, SSL, rate limits, and metrics per tenant
- **Dynamic Creation**: Create/update tenants via API without restart
- **Configuration Inheritance**: Tenants can inherit from parent configurations
- **Domain-based Routing**: Route requests based on domain

### Configuration

In your JSON config:

```json
{
  "apps": {
    "tenants": {
      "api_endpoint": "/api/tenants",
      "default_tenant": {
        "routes": [...],
        "rate_limit": {"rps": 100}
      },
      "tenants": {
        "tenant1": {
          "domains": ["tenant1.com"],
          "inherits": "default",
          "tls": {...}
        }
      }
    }
  }
}
```

### API Endpoints

- `GET /api/tenants` - List all tenants
- `POST /api/tenants` - Create a new tenant
- `GET /api/tenants/{name}` - Get tenant details
- `PUT /api/tenants/{name}` - Update tenant
- `DELETE /api/tenants/{name}` - Delete tenant

### Tenant Properties

- **Domains**: Domain names for the tenant
- **Routes**: HTTP routes and handlers
- **TLS**: SSL certificate configuration
- **Rate Limit**: Request rate limiting
- **Metrics**: Tenant-specific metrics collection

## Advanced Caching with Predictive Warming

ProxFlow includes an advanced caching layer with intelligent features for optimal performance.

### Features

- **Intelligent Caching**: Cache API responses to reduce backend load
- **Predictive Cache Warming**: Analyze access patterns to preload frequently accessed data
- **Cache Invalidation via Webhooks**: External systems can invalidate cache entries
- **Partial Cache Updates**: Update specific parts of cached JSON data
- **Distributed Cache Synchronization**: Keep cache consistent across multiple instances
- **Configurable TTL and Size Limits**: Fine-tune cache behavior

### Configuration

In your Caddyfile:

```
cache {
    ttl 10m
    max_size 100
    predictive_warming
    webhook_path /cache/invalidate
    distributed_sync {
        peer http://peer1:2019
        peer http://peer2:2019
        sync_interval 30s
    }
}
```

### Cache Invalidation Webhook

Send POST requests to `/cache/invalidate` with JSON payload:

```json
{
  "action": "invalidate",
  "key": "cache_key"
}
```

Or invalidate by pattern:

```json
{
  "action": "invalidate",
  "pattern": "api/users"
}
```

### Partial Updates

Update specific JSON fields without invalidating the entire cache entry:

```json
{
  "action": "partial_update",
  "partial": {
    "key": "user_123",
    "path": "user.email",
    "value": "newemail@example.com"
  }
}
```

### Parameters

- `ttl`: Default time-to-live for cache entries (default 5m)
- `max_size`: Maximum cache size in MB (default 100)
- `predictive_warming`: Enable predictive cache warming
- `webhook_path`: Path for cache invalidation webhooks (default /cache/invalidate)
- `distributed_sync`: Configuration for multi-instance synchronization

## Advanced Rate Limiting

ProxFlow includes advanced rate limiting with multiple algorithms and distributed support.

### Supported Algorithms

- **Token Bucket**: Smooth rate limiting with burst capacity
- **Leaky Bucket**: Request processing at a constant rate
- **Sliding Window**: Time-based request counting

### Rate Limiting Keys

- **IP Address**: Rate limit by client IP
- **User ID**: Rate limit by authenticated user
- **API Key**: Rate limit by API key
- **Custom Headers**: Rate limit by any HTTP header

### Configuration

In your Caddyfile:

```
rate_limit {
    algorithm token_bucket
    rate 10
    burst 5
    key ip
    reject 429
    distributed {
        redis {
            address localhost:6379
            password secret
            db 0
        }
        sync_interval 30s
    }
}
```

### Distributed Rate Limiting

For multi-instance deployments, configure Redis-backed distributed rate limiting:

```caddyfile
rate_limit {
    distributed {
        redis {
            address redis-cluster:6379
            password mypassword
            db 1
        }
        sync_interval 15s
    }
}
```

### Rejection Policies

- **429**: Return HTTP 429 Too Many Requests
- **delay**: Add delay before processing
- **drop**: Silently drop the request

## Request Replay and Traffic Shadowing

ProxFlow supports traffic shadowing and replay for testing and monitoring.

### Traffic Shadowing

Capture production traffic and mirror it to staging/test environments:

```caddyfile
traffic_shadow {
    sampling_rate 0.1
    compare_responses
    log_differences
    target staging http://staging.example.com/api {
        header X-Shadow true
        timeout 10s
        ignore_errors
    }
}
```

### Request Replay

Store and replay captured requests for testing:

```caddyfile
traffic_shadow {
    replay {
        storage {
            type redis
            redis {
                address localhost:6379
                db 1
            }
        }
        replay_rate 0.5
        max_replays 5
    }
}
```

### Response Comparison

Automatically compare responses between production and shadow environments to detect regressions:

- Status code comparison
- Response body comparison
- Performance timing analysis
- Configurable similarity thresholds

### Storage Options

- **Memory**: In-memory storage for development
- **File**: File-based storage for persistence
- **Redis**: Distributed storage for multi-instance deployments

## Auto-Scaling Backend Pool Management

ProxFlow includes intelligent auto-scaling backend pool management with service discovery integration.

### Features

- **Automatic Scaling**: Add/remove backend instances based on load and health
- **Multi-Platform Discovery**: Kubernetes, Docker Swarm, and Consul integration
- **Health Monitoring**: Continuous health checks with customizable thresholds
- **Load-Based Scaling**: Scale up/down based on backend utilization
- **Custom Health Checks**: Per-backend health check configuration

### Service Discovery

#### Kubernetes

```caddyfile
reverse_proxy {
    dynamic_upstreams auto_scaling {
        service_discovery {
            type kubernetes
            kubernetes {
                namespace default
                service_name my-app
                label_selector app=my-app
            }
        }
    }
}
```

#### Docker Swarm

```caddyfile
reverse_proxy {
    dynamic_upstreams auto_scaling {
        service_discovery {
            type docker_swarm
            docker_swarm {
                service_name my-app
                network_name my-network
            }
        }
    }
}
```

#### Consul

```caddyfile
reverse_proxy {
    dynamic_upstreams auto_scaling {
        service_discovery {
            type consul
            consul {
                service_name my-app
                address consul-server:8500
                token my-token
            }
        }
    }
}
```

### Health Checks

Configure health check parameters:

```caddyfile
reverse_proxy {
    dynamic_upstreams auto_scaling {
        health_checks {
            path /health
            interval 30s
            timeout 5s
            unhealthy_threshold 3
            healthy_threshold 2
        }
    }
}
```

### Auto-Scaling Configuration

```caddyfile
reverse_proxy {
    dynamic_upstreams auto_scaling {
        scaling {
            min_backends 1
            max_backends 10
            scale_up_threshold 0.8
            scale_down_threshold 0.2
            scale_cooldown 5m
            provider kubernetes
        }
    }
}
```

### Scaling Triggers

- **Scale Up**: When average backend load exceeds `scale_up_threshold`
- **Scale Down**: When average backend load falls below `scale_down_threshold`
- **Health-Based**: Remove unhealthy backends, add healthy ones
- **Cooldown**: Prevent rapid scaling operations

### Supported Providers

- **Kubernetes**: Horizontal Pod Autoscaling (HPA) integration
- **Docker Swarm**: Service scaling via Docker API
- **Manual**: Static backend management

## Zero Downtime Certificate Rotation

ProxFlow supports zero downtime certificate rotation with multiple certificate authorities and advanced TLS features.

### Certificate Authorities

- **Let's Encrypt**: Automatic ACME certificates
- **ZeroSSL**: Alternative ACME provider
- **Custom CA**: Support for internal certificate authorities

### Advanced TLS Features

- **OCSP Stapling**: Automatic OCSP response stapling for improved performance
- **Encrypted SNI (ECH)**: Encrypted Client Hello for enhanced privacy
- **Certificate Pinning**: Pin certificates to prevent man-in-the-middle attacks

### Configuration

```caddyfile
tls {
    protocols tls1.2 tls1.3
    ciphers ECDHE-ECDSA-AES128-GCM-SHA256
    curves x25519
}
```

## gRPC Gateway with Protocol Translation

ProxFlow includes a bidirectional gRPC gateway that translates between HTTP/REST and gRPC protocols.

### Features

- **Protocol Translation**: Automatic conversion between HTTP JSON and gRPC protobuf
- **Bidirectional Communication**: Support for both request and response translation
- **Streaming Support**: Handle gRPC streaming methods
- **gRPC-Web Compatibility**: Support for web browser gRPC calls
- **Built-in Reflection**: Automatic service discovery and method introspection

### Configuration

In your Caddyfile:

```
grpc_gateway localhost:50051 {
    proto_file /path/to/service.proto
    service user {
        method GET UserService/GetUser
        method POST UserService/CreateUser
        streaming
    }
    enable_reflection
    enable_grpc_web
    allow_cors
}
```

### Usage

The gateway translates REST API calls to gRPC:

```bash
# REST call
curl -X POST http://gateway/user/create \
  -H "Content-Type: application/json" \
  -d '{"name": "John", "email": "john@example.com"}'

# Becomes gRPC call to UserService.CreateUser
```

### Protocol Translation

- **Request**: JSON → Protobuf
- **Response**: Protobuf → JSON
- **Headers**: HTTP headers ↔ gRPC metadata
- **Status Codes**: HTTP status ↔ gRPC status codes

## [Features](https://caddyserver.com/features)

ProxFlow includes a GraphQL Federation Gateway that can federate multiple GraphQL services into a unified API endpoint. This is particularly useful in microservices architectures where each service exposes its own GraphQL API.

### Features

- **Schema Stitching**: Automatically merges schemas from multiple GraphQL services.
- **Intelligent Query Planning**: Plans queries across services for efficient execution.
- **Caching**: Caches query plans and responses to improve performance.
- **Complexity Limiting**: Prevents abuse by limiting query complexity.
- **Introspection Support**: Optional support for GraphQL introspection.

### Configuration

In your Caddyfile:

```
graphql_federation {
    service user_service http://user-service:4001/graphql {
        schema "...user schema SDL..."
    }
    service product_service http://product-service:4002/graphql {
        schema "...product schema SDL..."
    }
    cache_duration 5m
    max_query_complexity 1000
    enable_introspection
}
```

- `service`: Defines an upstream GraphQL service with name, URL, and schema.
- `cache_duration`: How long to cache plans and responses (default 5m).
- `max_query_complexity`: Maximum allowed query complexity (default 1000).
- `enable_introspection`: Allows GraphQL introspection queries.

### Usage

The gateway exposes a single GraphQL endpoint that can query across all federated services. For example, if you have user and product services, you can query both in a single request.

## Real-Time Traffic Visualization Dashboard

ProxFlow includes a web-based dashboard for real-time traffic monitoring and visualization.

### Features

- **Real-Time Metrics**: Request rates, error rates, and active connections
- **Latency Monitoring**: P50, P95, P99 latency percentiles
- **Geographic Distribution**: Requests by region
- **Live Updates**: Server-Sent Events (SSE) for real-time data streaming
- **Interactive Dashboard**: Web interface for monitoring traffic patterns

### Configuration

In your Caddyfile:

```
traffic_dashboard {
    path /dashboard
    update_interval 1s
}
```

- `path`: The URL path where the dashboard is served (default `/dashboard`)
- `update_interval`: How often to send updates (default 1s)

### Usage

Navigate to the configured path (e.g., `http://your-server/dashboard`) to view the dashboard. The page will automatically update with real-time traffic data via Server-Sent Events.

## Intelligent Circuit Breaker with Auto Recovery

ProxFlow includes an intelligent circuit breaker that monitors backend health using multiple metrics and provides automatic recovery with traffic ramp-up.

### Features

- **Multi-Metric Monitoring**: Tracks latency, error rates, and timeouts
- **Automatic Circuit Management**: Opens/closes circuits based on configurable thresholds
- **Smart Recovery**: Half-open state for testing recovery, followed by ramp-up phase
- **Traffic Ramp-Up**: Gradual traffic increase during recovery to prevent backend overload
- **Configurable Parameters**: Thresholds, timeouts, and ramp-up durations

### Configuration

In your Caddyfile, configure per upstream:

```
reverse_proxy localhost:8080 {
    circuit_breaker intelligent {
        failure_threshold 0.5
        recovery_timeout 60s
        half_open_max_requests 3
        latency_threshold 5s
        min_requests 10
        evaluation_window 60s
        ramp_up_duration 30s
    }
}
```

### Parameters

- `failure_threshold`: Failure rate (0-1) to trigger circuit open (default 0.5)
- `recovery_timeout`: Time to wait before attempting recovery (default 60s)
- `half_open_max_requests`: Max requests in half-open state (default 3)
- `latency_threshold`: Latency threshold for slow requests (default 5s)
- `min_requests`: Minimum requests needed for evaluation (default 10)
- `evaluation_window`: Time window for metric evaluation (default 60s)
- `ramp_up_duration`: Duration of traffic ramp-up phase (default 30s)

### Circuit States

1. **Closed**: Normal operation, all requests pass through
2. **Open**: Circuit is open, requests are blocked
3. **Half-Open**: Testing recovery with limited requests
4. **Ramp-Up**: Gradual traffic increase during recovery

## API Gateway with Request Transformation

ProxFlow includes an API gateway with powerful request/response transformation capabilities.

### Features

- **Request/Response Transformation**: Modify headers, body, and structure
- **Request Enrichment**: Add additional data from external sources
- **Response Aggregation**: Combine responses from multiple backends
- **Flexible Matching**: Apply transformations based on request conditions

### Configuration

In your Caddyfile:

```
transform {
	header add X-API-Version "v1"
	body replace user.name "enriched"
	enrich {
		data key1 value1
		from_url https://api.example.com/enrich
	}
	aggregate {
		backend https://backend1.com/api
		backend https://backend2.com/api
		combine merge
	}
}
```

### Transformation Types

- **Header Actions**: add, set, delete headers
- **Body Actions**: replace, append, prepend JSON values
- **Enrichment**: Add data from external APIs
- **Aggregation**: Combine multiple backend responses

## Multi-Tenancy with Dynamic Routing

ProxFlow supports multi-tenant architectures with isolated configurations and dynamic tenant management.

### Features

- **Tenant Isolation**: Separate routing, SSL, rate limits, and metrics per tenant
- **Dynamic Creation**: Create/update tenants via API without restart
- **Configuration Inheritance**: Tenants can inherit from parent configurations
- **Domain-based Routing**: Route requests based on domain

### Configuration

In your JSON config:

```json
{
  "apps": {
    "tenants": {
      "api_endpoint": "/api/tenants",
      "default_tenant": {
        "routes": [...],
        "rate_limit": {"rps": 100}
      },
      "tenants": {
        "tenant1": {
          "domains": ["tenant1.com"],
          "inherits": "default",
          "tls": {...}
        }
      }
    }
  }
}
```

### API Endpoints

- `GET /api/tenants` - List all tenants
- `POST /api/tenants` - Create a new tenant
- `GET /api/tenants/{name}` - Get tenant details
- `PUT /api/tenants/{name}` - Update tenant
- `DELETE /api/tenants/{name}` - Delete tenant

### Tenant Properties

- **Domains**: Domain names for the tenant
- **Routes**: HTTP routes and handlers
- **TLS**: SSL certificate configuration
- **Rate Limit**: Request rate limiting
- **Metrics**: Tenant-specific metrics collection

## Advanced Caching with Predictive Warming

ProxFlow includes an advanced caching layer with intelligent features for optimal performance.

### Features

- **Intelligent Caching**: Cache API responses to reduce backend load
- **Predictive Cache Warming**: Analyze access patterns to preload frequently accessed data
- **Cache Invalidation via Webhooks**: External systems can invalidate cache entries
- **Partial Cache Updates**: Update specific parts of cached JSON data
- **Distributed Cache Synchronization**: Keep cache consistent across multiple instances
- **Configurable TTL and Size Limits**: Fine-tune cache behavior

### Configuration

In your Caddyfile:

```
cache {
    ttl 10m
    max_size 100
    predictive_warming
    webhook_path /cache/invalidate
    distributed_sync {
        peer http://peer1:2019
        peer http://peer2:2019
        sync_interval 30s
    }
}
```

### Cache Invalidation Webhook

Send POST requests to `/cache/invalidate` with JSON payload:

```json
{
  "action": "invalidate",
  "key": "cache_key"
}
```

Or invalidate by pattern:

```json
{
  "action": "invalidate",
  "pattern": "api/users"
}
```

### Partial Updates

Update specific JSON fields without invalidating the entire cache entry:

```json
{
  "action": "partial_update",
  "partial": {
    "key": "user_123",
    "path": "user.email",
    "value": "newemail@example.com"
  }
}
```

### Parameters

- `ttl`: Default time-to-live for cache entries (default 5m)
- `max_size`: Maximum cache size in MB (default 100)
- `predictive_warming`: Enable predictive cache warming
- `webhook_path`: Path for cache invalidation webhooks (default /cache/invalidate)
- `distributed_sync`: Configuration for multi-instance synchronization

## Advanced Rate Limiting

ProxFlow includes advanced rate limiting with multiple algorithms and distributed support.

### Supported Algorithms

- **Token Bucket**: Smooth rate limiting with burst capacity
- **Leaky Bucket**: Request processing at a constant rate
- **Sliding Window**: Time-based request counting

### Rate Limiting Keys

- **IP Address**: Rate limit by client IP
- **User ID**: Rate limit by authenticated user
- **API Key**: Rate limit by API key
- **Custom Headers**: Rate limit by any HTTP header

### Configuration

In your Caddyfile:

```
rate_limit {
    algorithm token_bucket
    rate 10
    burst 5
    key ip
    reject 429
    distributed {
        redis {
            address localhost:6379
            password secret
            db 0
        }
        sync_interval 30s
    }
}
```

### Distributed Rate Limiting

For multi-instance deployments, configure Redis-backed distributed rate limiting:

```caddyfile
rate_limit {
    distributed {
        redis {
            address redis-cluster:6379
            password mypassword
            db 1
        }
        sync_interval 15s
    }
}
```

### Rejection Policies

- **429**: Return HTTP 429 Too Many Requests
- **delay**: Add delay before processing
- **drop**: Silently drop the request

## Request Replay and Traffic Shadowing

ProxFlow supports traffic shadowing and replay for testing and monitoring.

### Traffic Shadowing

Capture production traffic and mirror it to staging/test environments:

```caddyfile
traffic_shadow {
    sampling_rate 0.1
    compare_responses
    log_differences
    target staging http://staging.example.com/api {
        header X-Shadow true
        timeout 10s
        ignore_errors
    }
}
```

### Request Replay

Store and replay captured requests for testing:

```caddyfile
traffic_shadow {
    replay {
        storage {
            type redis
            redis {
                address localhost:6379
                db 1
            }
        }
        replay_rate 0.5
        max_replays 5
    }
}
```

### Response Comparison

Automatically compare responses between production and shadow environments to detect regressions:

- Status code comparison
- Response body comparison
- Performance timing analysis
- Configurable similarity thresholds

### Storage Options

- **Memory**: In-memory storage for development
- **File**: File-based storage for persistence
- **Redis**: Distributed storage for multi-instance deployments

## Auto-Scaling Backend Pool Management

ProxFlow includes intelligent auto-scaling backend pool management with service discovery integration.

### Features

- **Automatic Scaling**: Add/remove backend instances based on load and health
- **Multi-Platform Discovery**: Kubernetes, Docker Swarm, and Consul integration
- **Health Monitoring**: Continuous health checks with customizable thresholds
- **Load-Based Scaling**: Scale up/down based on backend utilization
- **Custom Health Checks**: Per-backend health check configuration

### Service Discovery

#### Kubernetes

```caddyfile
reverse_proxy {
    dynamic_upstreams auto_scaling {
        service_discovery {
            type kubernetes
            kubernetes {
                namespace default
                service_name my-app
                label_selector app=my-app
            }
        }
    }
}
```

#### Docker Swarm

```caddyfile
reverse_proxy {
    dynamic_upstreams auto_scaling {
        service_discovery {
            type docker_swarm
            docker_swarm {
                service_name my-app
                network_name my-network
            }
        }
    }
}
```

#### Consul

```caddyfile
reverse_proxy {
    dynamic_upstreams auto_scaling {
        service_discovery {
            type consul
            consul {
                service_name my-app
                address consul-server:8500
                token my-token
            }
        }
    }
}
```

### Health Checks

Configure health check parameters:

```caddyfile
reverse_proxy {
    dynamic_upstreams auto_scaling {
        health_checks {
            path /health
            interval 30s
            timeout 5s
            unhealthy_threshold 3
            healthy_threshold 2
        }
    }
}
```

### Auto-Scaling Configuration

```caddyfile
reverse_proxy {
    dynamic_upstreams auto_scaling {
        scaling {
            min_backends 1
            max_backends 10
            scale_up_threshold 0.8
            scale_down_threshold 0.2
            scale_cooldown 5m
            provider kubernetes
        }
    }
}
```

### Scaling Triggers

- **Scale Up**: When average backend load exceeds `scale_up_threshold`
- **Scale Down**: When average backend load falls below `scale_down_threshold`
- **Health-Based**: Remove unhealthy backends, add healthy ones
- **Cooldown**: Prevent rapid scaling operations

### Supported Providers

- **Kubernetes**: Horizontal Pod Autoscaling (HPA) integration
- **Docker Swarm**: Service scaling via Docker API
- **Manual**: Static backend management

## Zero Downtime Certificate Rotation

ProxFlow supports zero downtime certificate rotation with multiple certificate authorities and advanced TLS features.

### Certificate Authorities

- **Let's Encrypt**: Automatic ACME certificates
- **ZeroSSL**: Alternative ACME provider
- **Custom CA**: Support for internal certificate authorities

### Advanced TLS Features

- **OCSP Stapling**: Automatic OCSP response stapling for improved performance
- **Encrypted SNI (ECH)**: Encrypted Client Hello for enhanced privacy
- **Certificate Pinning**: Pin certificates to prevent man-in-the-middle attacks

### Configuration

```caddyfile
tls {
    protocols tls1.2 tls1.3
    ciphers ECDHE-ECDSA-AES128-GCM-SHA256
    curves x25519
}
```

## gRPC Gateway with Protocol Translation

ProxFlow includes a bidirectional gRPC gateway that translates between HTTP/REST and gRPC protocols.

### Features

- **Protocol Translation**: Automatic conversion between HTTP JSON and gRPC protobuf
- **Bidirectional Communication**: Support for both request and response translation
- **Streaming Support**: Handle gRPC streaming methods
- **gRPC-Web Compatibility**: Support for web browser gRPC calls
- **Built-in Reflection**: Automatic service discovery and method introspection

### Configuration

In your Caddyfile:

```
grpc_gateway localhost:50051 {
    proto_file /path/to/service.proto
    service user {
        method GET UserService/GetUser
        method POST UserService/CreateUser
        streaming
    }
    enable_reflection
    enable_grpc_web
    allow_cors
}
```

### Usage

The gateway translates REST API calls to gRPC:

```bash
# REST call
curl -X POST http://gateway/user/create \
  -H "Content-Type: application/json" \
  -d '{"name": "John", "email": "john@example.com"}'

# Becomes gRPC call to UserService.CreateUser
```

### Protocol Translation

- **Request**: JSON → Protobuf
- **Response**: Protobuf → JSON
- **Headers**: HTTP headers ↔ gRPC metadata
- **Status Codes**: HTTP status ↔ gRPC status codes

## [Features](https://caddyserver.com/features)

- **Easy configuration** with the [Caddyfile](https://caddyserver.com/docs/caddyfile)
- **Powerful configuration** with its [native JSON config](https://caddyserver.com/docs/json/)
- **Dynamic configuration** with the [JSON API](https://caddyserver.com/docs/api)
- [**Config adapters**](https://caddyserver.com/docs/config-adapters) if you don't like JSON
- **Automatic HTTPS** by default
	- [ZeroSSL](https://zerossl.com) and [Let's Encrypt](https://letsencrypt.org) for public names
	- Fully-managed local CA for internal names & IPs
	- Can coordinate with other Caddy instances in a cluster
	- Multi-issuer fallback
	- Encrypted ClientHello (ECH) support
- **Stays up when other servers go down** due to TLS/OCSP/certificate-related issues
- **Production-ready** after serving trillions of requests and managing millions of TLS certificates
- **Scales to hundreds of thousands of sites** as proven in production
- **HTTP/1.1, HTTP/2, and HTTP/3** all supported by default
- **Highly extensible** [modular architecture](https://caddyserver.com/docs/architecture) lets Caddy do anything without bloat
- **Runs anywhere** with **no external dependencies** (not even libc)
- Written in Go, a language with higher **memory safety guarantees** than other servers
- Actually **fun to use**
- So much more to [discover](https://caddyserver.com/features)

## Install

The simplest, cross-platform way to get started is to download Caddy from [GitHub Releases](https://github.com/caddyserver/caddy/releases) and place the executable file in your PATH.

See [our online documentation](https://caddyserver.com/docs/install) for other install instructions.

## Build from source

Requirements:

- [Go 1.25.0 or newer](https://golang.org/dl/)

### For development

_**Note:** These steps [will not embed proper version information](https://github.com/golang/go/issues/29228). For that, please follow the instructions in the next section._

```bash
$ git clone "https://github.com/caddyserver/caddy.git"
$ cd caddy/cmd/caddy/
$ go build
```

When you run Caddy, it may try to bind to low ports unless otherwise specified in your config. If your OS requires elevated privileges for this, you will need to give your new binary permission to do so. On Linux, this can be done easily with: `sudo setcap cap_net_bind_service=+ep ./caddy`

If you prefer to use `go run` which only creates temporary binaries, you can still do this with the included `setcap.sh` like so:

```bash
$ go run -exec ./setcap.sh main.go
```

If you don't want to type your password for `setcap`, use `sudo visudo` to edit your sudoers file and allow your user account to run that command without a password, for example:

```
username ALL=(ALL:ALL) NOPASSWD: /usr/sbin/setcap
```

replacing `username` with your actual username. Please be careful and only do this if you know what you are doing! We are only qualified to document how to use Caddy, not Go tooling or your computer, and we are providing these instructions for convenience only; please learn how to use your own computer at your own risk and make any needful adjustments.

Then you can run the tests in all modules or a specific one:

```bash
$ go test ./...
$ go test ./modules/caddyhttp/tracing/
```

### With version information and/or plugins

Using [our builder tool, `xcaddy`](https://github.com/caddyserver/xcaddy)...

```bash
$ xcaddy build
```

...the following steps are automated:

1. Create a new folder: `mkdir caddy`
2. Change into it: `cd caddy`
3. Copy [Caddy's main.go](https://github.com/caddyserver/caddy/blob/master/cmd/caddy/main.go) into the empty folder. Add imports for any custom plugins you want to add.
4. Initialize a Go module: `go mod init caddy`
5. (Optional) Pin Caddy version: `go get github.com/caddyserver/caddy/v2@version` replacing `version` with a git tag, commit, or branch name.
6. (Optional) Add plugins by adding their import: `_ "import/path/here"`
7. Compile: `go build -tags=nobadger,nomysql,nopgx`




## Quick start

The [Caddy website](https://caddyserver.com/docs/) has documentation that includes tutorials, quick-start guides, reference, and more.

**We recommend that all users -- regardless of experience level -- do our [Getting Started](https://caddyserver.com/docs/getting-started) guide to become familiar with using Caddy.**

If you've only got a minute, [the website has several quick-start tutorials](https://caddyserver.com/docs/quick-starts) to choose from! However, after finishing a quick-start tutorial, please read more documentation to understand how the software works. 🙂




## Overview

Caddy is most often used as an HTTPS server, but it is suitable for any long-running Go program. First and foremost, it is a platform to run Go applications. Caddy "apps" are just Go programs that are implemented as Caddy modules. Two apps -- `tls` and `http` -- ship standard with Caddy.

Caddy apps instantly benefit from [automated documentation](https://caddyserver.com/docs/json/), graceful on-line [config changes via API](https://caddyserver.com/docs/api), and unification with other Caddy apps.

Although [JSON](https://caddyserver.com/docs/json/) is Caddy's native config language, Caddy can accept input from [config adapters](https://caddyserver.com/docs/config-adapters) which can essentially convert any config format of your choice into JSON: Caddyfile, JSON 5, YAML, TOML, NGINX config, and more.

The primary way to configure Caddy is through [its API](https://caddyserver.com/docs/api), but if you prefer config files, the [command-line interface](https://caddyserver.com/docs/command-line) supports those too.

Caddy exposes an unprecedented level of control compared to any web server in existence. In Caddy, you are usually setting the actual values of the initialized types in memory that power everything from your HTTP handlers and TLS handshakes to your storage medium. Caddy is also ridiculously extensible, with a powerful plugin system that makes vast improvements over other web servers.

To wield the power of this design, you need to know how the config document is structured. Please see [our documentation site](https://caddyserver.com/docs/) for details about [Caddy's config structure](https://caddyserver.com/docs/json/).

Nearly all of Caddy's configuration is contained in a single config document, rather than being scattered across CLI flags and env variables and a configuration file as with other web servers. This makes managing your server config more straightforward and reduces hidden variables/factors.


## Full documentation

Our website has complete documentation:

**https://caddyserver.com/docs/**

The docs are also open source. You can contribute to them here: https://github.com/caddyserver/website



## Getting help

- We advise companies using Caddy to secure a support contract through [Ardan Labs](https://www.ardanlabs.com) before help is needed.

- A [sponsorship](https://github.com/sponsors/mholt) goes a long way! We can offer private help to sponsors. If Caddy is benefitting your company, please consider a sponsorship. This not only helps fund full-time work to ensure the longevity of the project, it provides your company the resources, support, and discounts you need; along with being a great look for your company to your customers and potential customers!

- Individuals can exchange help for free on our community forum at https://caddy.community. Remember that people give help out of their spare time and good will. The best way to get help is to give it first!

Please use our [issue tracker](https://github.com/caddyserver/caddy/issues) only for bug reports and feature requests, i.e. actionable development items (support questions will usually be referred to the forums).



## About

Matthew Holt began developing Caddy in 2014 while studying computer science at Brigham Young University. (The name "Caddy" was chosen because this software helps with the tedious, mundane tasks of serving the Web, and is also a single place for multiple things to be organized together.) It soon became the first web server to use HTTPS automatically and by default, and now has hundreds of contributors and has served trillions of HTTPS requests.

**The name "Caddy" is trademarked.** The name of the software is "Caddy", not "Caddy Server" or "CaddyServer". Please call it "Caddy" or, if you wish to clarify, "the Caddy web server". Caddy is a registered trademark of Stack Holdings GmbH.

- _Project on X: [@caddyserver](https://x.com/caddyserver)_
- _Author on X: [@mholt6](https://x.com/mholt6)_

Caddy is a project of [ZeroSSL](https://zerossl.com), a Stack Holdings company.

Debian package repository hosting is graciously provided by [Cloudsmith](https://cloudsmith.com). Cloudsmith is the only fully hosted, cloud-native, universal package management solution, that enables your organization to create, store and share packages in any format, to any place, with total confidence.
