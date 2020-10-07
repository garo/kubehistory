FROM golang:1.11 AS builder
RUN mkdir /spark-driver-proxy
WORKDIR /spark-driver-proxy
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go test
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /app .

FROM scratch
COPY --from=builder /app ./
ADD pages pages/
ADD static static/
ADD migrations migrations/
ENTRYPOINT ["./app"]
