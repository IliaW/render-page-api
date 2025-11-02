## Render Page API

A Go-based web service that makes requests to the web pages and verifies whether they were rendered using headless
Chrome browsers. Captures full screenshots if enabled. The service uses a browser pool to handle concurrent requests.

### How to run

- Take a look at `config.yaml`
- Run `docker-compose up`

### API Endpoints

**GET /api/render?url=https://some-url.com**

<pre>{
    "url": "https://some-url.com",
    "rendering": "success",
    "status_code": 200
}</pre>

**GET /ping**
<pre>{
    "message": "pong"
}</pre>

**NOTE:**

- `/api/render` path is configurable in `config.yaml`
- `/api/render` changes `http` prefix to `https` or adds the last one if it is missed