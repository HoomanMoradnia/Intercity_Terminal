# Docker Setup Instructions

This repository contains a Go-based Transportation Portal application with a PostgreSQL database backend. The application is containerized using Docker for easy deployment.

## Prerequisites

- Docker and Docker Compose installed on your system
- Make sure ports 5000 and 5432 are available on your machine

## Quick Start

1. Run the helper script to handle all setup and start the application:

```
./run-docker.sh
```

This script will:
- Clean up any existing containers
- Create the vendor directory if needed
- Build and start the application containers

2. Access the application at: http://localhost:5000

## Manual Docker Commands

If you prefer to run the commands manually:

1. First, create the vendor directory with all dependencies:

```
./prepare-build.sh
```

2. Build and start the containers:

```
sudo docker compose up --build
```

3. To run in detached mode:

```
sudo docker compose up --build -d
```

4. To view logs when running in detached mode:

```
sudo docker compose logs -f
```

5. To stop the containers:

```
sudo docker compose down
```

## Troubleshooting

If you encounter network-related issues during the build process:

1. The vendor directory includes all dependencies to avoid network calls during build
2. DNS servers are set to Google's public DNS (8.8.8.8 and 8.8.4.4)
3. Network-related Go environment variables are set to minimize external connections

If you still have issues:
- Check your system's network configuration
- Verify that Docker has network access
- Make sure all dependencies are correctly vendored

## Environment Variables

The application uses the following environment variables, which are set in the `.env` file:

- `PGHOST`: PostgreSQL host address (default: db)
- `PGPORT`: PostgreSQL port (default: 5432)
- `PGUSER`: PostgreSQL username (default: postgres)
- `PGPASSWORD`: PostgreSQL password (default: postgres)
- `PGDATABASE`: PostgreSQL database name (default: transport)
- `DATABASE_URL`: Full PostgreSQL connection string