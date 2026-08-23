# Deploy

## Server setup
Rent a VDS server. Generate SSH keys and copy your public key to the server.

## Reverse proxy
Although the application ensures it runs in a safe environment, it is not responsible for rate limiting, HTTPS and high-level routing. Therefore, without a reverse proxy, it is inaccessible from the internet.

Configure any reverse proxy. Below is an example configuration using Caddy:

```caddyfile
{
	servers {
		timeouts {
			idle 2m
		}
	}
}

goroutine.mipselqq.uk {
	reverse_proxy localhost:8080 {
		transport http {
			read_timeout  5s
			write_timeout 10s
		}
	}
	request_body {
		max_size 20KB
	}
}

grafana.goroutine.mipselqq.uk {
	reverse_proxy localhost:3000
	request_body {
		max_size 10MB
	}
}

prometheus.goroutine.mipselqq.uk {
	reverse_proxy localhost:9090
	request_body {
		max_size 1MB
	}
}
```

## Configuration
Variable and secret settings are located in your repository under **Settings > Secrets and variables > Actions**.

<img width="1506" height="1431" alt="github variables page" src="https://github.com/user-attachments/assets/8e2b8e9b-d3eb-4f60-86de-0849396c0ae4" />

> [!NOTE]
> Any variable or secret can be set repository-wide for both environments, or specified per environment (e.g., `staging` or `production`).

### Variables
- `DEPLOY_HOST`: Target server hostname or IP (`goroutine.mipselqq.uk`)
- `DEPLOY_USERNAME`: SSH user for deployment (`deployer`)
- `ADMIN_PORT`: Port for administrative endpoints (`9091`) — written to `.env.prod` as `APP_ADMIN_PORT`
- `ALLOWED_ORIGINS`: Allowed CORS origins — `APP_ALLOWED_ORIGINS`
- `ENV`: Runtime environment — `APP_ENV`
- `HOST`: Server bind address — `APP_HOST`
- `JWT_EXP`: Token expiration — `APP_JWT_EXP`
- `LOG_LEVEL`: Application log level — `APP_LOG_LEVEL`
- `PORT`: Main application port — `APP_PORT`
- `POSTGRES_DB`: Database name (`todo_db`)
- `POSTGRES_HOST`: Database host (`db`)
- `POSTGRES_PORT`: Database port (`5432`)
- `POSTGRES_USER`: Database user (`user`)
- `PROMETHEUS_USER`: Prometheus dashboard username (`admin`)
- `REDIS_HOST`: Redis host (`redis`)
- `REDIS_PORT`: Redis port (`6379`)
- `TELEGRAM_LINK_TOKEN_TTL`: TTL for Telegram link tokens (`15m`)
- `SWAGGER_HOST`: API documentation host — `APP_SWAGGER_HOST`
- `NOTIFY_ENV`: Notifier runtime environment (`dev` / `prod`)
- `NOTIFY_LOG_LEVEL`: Notifier log level (`info`)

### Secrets
- `DEPLOY_SSH_KEY`: Private SSH key for server access
- `GF_SECURITY_ADMIN_PASSWORD`: Admin password for Grafana
- `JWT_SECRET`: Secret key used for JWT signing — written to `.env.prod` as `APP_JWT_SECRET`
- `POSTGRES_PASSWORD`: Database password
- `PROMETHEUS_BCRYPT_HASH`: Bcrypt hash of the Prometheus password
- `PROMETHEUS_PASSWORD`: Plain password for Prometheus
- `REDIS_PASSWORD`: Redis password
- `TELEGRAM_BOT_TOKEN`: Telegram bot token from @BotFather

## Continuous deployment
After the deploy action is triggered, the application will be automatically built, transferred, and run on the server.
