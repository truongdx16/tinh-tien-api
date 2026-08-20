# OpenAPI — generate & use

This project uses **contract-first OpenAPI**: the spec lives in [`api/openapi.yaml`](../api/openapi.yaml) and is maintained by hand to match the Go handlers.

It is **not** auto-generated from Go code today. When you add or change endpoints, update `api/openapi.yaml` too.

---

## 1. Browse & test (Swagger UI)

Start the API from the **project root** (so `api/openapi.yaml` is found):

```bash
make docker-up
make run
```

Open in browser:

| URL                                  | Purpose    |
| ------------------------------------ | ---------- |
| <http://localhost:2170/docs>         | Swagger UI |
| <http://localhost:2170/openapi.yaml> | Raw spec   |

### Try an endpoint in Swagger UI

1. Expand **POST /v1/auth/login**
2. **Try it out** → body: `{"username":"owner","password":"owner123"}`
3. **Execute** → copy `token` from the response
4. Click **Authorize** (top right)
5. Paste the token (Swagger adds `Bearer` automatically)
6. Call other endpoints (e.g. **GET /v1/products**)

---

## 2. Import into Postman / Insomnia

1. **Import** → **Link** or **File**
2. Choose `api/openapi.yaml` or `http://localhost:2170/openapi.yaml`
3. Set collection auth: **Bearer Token**
4. Run login request first, set token on the collection

---

## 3. Validate the spec

With Docker (no install):

```bash
make openapi-validate
```

With npm (if you have Node.js):

```bash
npx @redocly/cli lint api/openapi.yaml
```

---

## 4. Generate client code (frontend / mobile)

Install [OpenAPI Generator](https://openapi-generator.tech) once, then generate:

### TypeScript (Axios) — for React/Vue frontend

```bash
make openapi-gen-ts
# output: clients/typescript/
```

### Other languages

```bash
docker run --rm -v "$PWD:/local" openapitools/openapi-generator-cli generate \
  -i /local/api/openapi.yaml \
  -g kotlin \
  -o /local/clients/kotlin
```

Common `-g` values: `typescript-axios`, `typescript-fetch`, `dart`, `kotlin`, `swift5`.

Generated clients call the same URLs as the spec (`http://localhost:2170` by default). Change `servers` in the YAML for staging/production.

---

## 5. Generate Go server stubs (optional)

If you want Go types/handlers **from** the YAML (codegen):

```bash
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

oapi-codegen -generate types,chi-server -package api \
  -o internal/generated/api.go api/openapi.yaml
```

This project already implements handlers manually in `internal/domain/*`. Use codegen when you want typed request/response structs shared with the frontend, or when rebuilding from scratch.

---

## 6. Auto-generate spec **from** Go (alternative approach)

Tools like [swaggo/swag](https://github.com/swaggo/swag) read comments on handlers and emit OpenAPI + Swagger UI:

```go
// @Summary List products
// @Security BearerAuth
// @Router /v1/products [get]
```

We did **not** use swag in this repo to keep handlers clean. You can adopt it later and replace the manual YAML.

---

## Workflow summary

```
api/openapi.yaml  ──►  Swagger UI (/docs)
                   ──►  Postman import
                   ──►  openapi-generator → frontend client
                   ──►  oapi-codegen → Go types (optional)

Go handlers (internal/domain)  ──►  implement the contract
```

**Rule of thumb:** change API → update YAML → regenerate client if you use codegen.
