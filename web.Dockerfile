# Build WASM
FROM golang:1.25-alpine AS gobuilder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOOS=js GOARCH=wasm go build -o dockadvisor.wasm wasm/wasm.go


FROM node:25-alpine AS nodebase


FROM nodebase AS deps
# Check https://github.com/nodejs/docker-node/tree/b4117f9333da4138b03a546ec926ef50a31506c3#nodealpine to understand why libc6-compat might be needed.
RUN apk add --no-cache libc6-compat
WORKDIR /app

# Install dependencies
COPY web/package.json web/package-lock.json* ./
RUN npm ci


# Rebuild the source code only when needed
FROM nodebase AS nodebuilder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY web/ .

RUN npm run build

# Production image
FROM nginx:1.29-alpine AS nginx

COPY --from=nodebuilder /app/dist/ /usr/share/nginx/html
COPY --from=gobuilder /build/dockadvisor.wasm /usr/share/nginx/html/js