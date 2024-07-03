# PasteAPI

## Documentation

Visit **https://zhukovrost.github.io/pasteAPI/** to see documentation

or visit **http://localhost:8080/swagger** while api is running.

**Note:** The documentation won't work properly if api in not running.

## Instructions for launching on a local computer

**Docker needed**. Docker installation instruction: https://docs.docker.com/get-docker/

### Step 1: Clone the repository

```sh
git clone https://github.com/zhukovrost/pasteAPI.git
```

### Step 2: Go to the app directory

```sh
cd pasteAPI
```

### Step 3 (optional): Edit .env file

You can edit .env file to configure the environment variables.
Visit internal/config/config.go to see all possible environment variables.

### Step 4: build the image

```sh
make build/image
```

**Note:** if the needed ports are used by other programs locally, you can run **make dev/services/stop**

### Step 5: Run the application using Docker

```sh
make run/api
```

or 

```sh
docker compose up
```

### Step 6: Healthcheck

Visit **http://localhost:8080/api/v1/healthcheck** to make sure everything is working correctly.
