FROM golang:1.21 as GOBUILD
WORKDIR /build
COPY ./cache/ ./cache
COPY ./*.go ./
COPY ./go.mod ./
RUN go get
RUN CGO_ENABLED=0 GOOS=linux go build -o server 

FROM node as REACTBUILD
WORKDIR /build
COPY ./react/public/ ./public/
COPY ./react/src/ ./src/
COPY ./react/package.json ./
COPY ./react/pnpm-lock.yaml ./
RUN corepack enable
RUN pnpm install
RUN pnpm build

FROM alpine
WORKDIR /app
COPY --from=GOBUILD /build/server ./
COPY --from=REACTBUILD /build/build/ ./build/
COPY ./.env ./
EXPOSE 5050
CMD ./server
