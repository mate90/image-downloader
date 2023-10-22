# Go Image Downloader and Resizer

This Go application downloads images from a specified search query, resizes them, and stores them in a PostgreSQL database.

## Prerequisites

- Go (version 1.13 or higher)
- PostgreSQL database
- Required Go packages: Colly, PostgreSQL driver, Resize

## Configuration

1. **Environment Variables**: Create a `.env` file with the following environment variables:

   ```makefile
   DB_HOST=<database_host>
   DB_PORT=<database_port>
   DB_USER=<database_username>
   DB_PASSWORD=<database_password>
   DB_NAME=<database_name>

## Usage
Input Configuration: Edit inputs.json to specify search queries and the number of images to download for each query.

- json
- Copy code
[
  {"SearchQuery": "cats", "MaxImages": 10},
  {"SearchQuery": "dogs", "MaxImages": 15}
]

## Run the Application:

- bash
- Copy code
- go run main.go
- The application will download images based on the search queries, resize them, and store them in the database.

## Dockerization
- To run the application in a Docker container:

- Build the Docker Image:

- bash
- Copy code
- docker build -t scalesops_task .
- Run the Docker Container:

- bash
- Copy code
- docker run -d --name scalesops_task_container scalesops_task
