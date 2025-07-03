# Money Track REST API

It provides a REST API for CRUD operations needed by MoneyTrack Application.

## Quick Start with Docker

### Prerequisites

-   Docker
-   Docker Compose

### Running the Application

1. Clone the repository
2. Navigate to the project directory
3. Run the application with Docker Compose:
    ```bash
    docker-compose up --build
    ```

The API will be available at `http://localhost:8080`

### Environment Variables

The application uses the following environment variables (defined in `.env`):

-   `DB_HOST`: Database host (default: mysql-db)
-   `DB_DRIVER`: Database driver (default: mysql)
-   `DB_USER`: Database user (default: root)
-   `DB_PASSWORD`: Database password
-   `DB_NAME`: Database name (default: moneytrack)
-   `DB_PORT`: Database port (default: 3306)
-   `API_SECRET`: JWT secret key
-   `TOKEN_HOUR_LIFESPAN`: JWT token lifespan in hours
-   `PORT`: Application port (default: 8080)

### Manual Development Setup

If you prefer to run without Docker:

1. Install Go 1.23+
2. Install MySQL
3. Update `.env` file with your database settings
4. Run: `go run main.go`

## Official Documentation

Documentation for the REST API can be found on the [MoneyTrack API](https://documenter.getpostman.com/view/4800685/S1LzwReP).

## Security Vulnerabilities

If you discover a security vulnerability, please send an e-mail to Panagiotis Dimopoulos at panosdim@gmail.com. All security vulnerabilities will be promptly addressed.

## License

The REST API is open-sourced software licensed under the [MIT license](https://opensource.org/licenses/MIT).
