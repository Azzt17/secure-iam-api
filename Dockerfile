FROM golang:1.25.10-alpine AS builder

WORKDIR /app

# Copy file dependensi
COPY go.mod go.sum ./

# Checksum
RUN go mod download && go mod verify

# Copy seluruh source code
COPY . .

# Kompilasi Binary:
# - CGO_ENABLED=0 : agar binary bersifat statis (tidak butuh library C bawaan OS), syarat utama untuk distroless
# - GOOS=linux & GOARCH=amd64 : memastikan target OS dan arsitektur CPU
# - -ldflags="-s -w" : menghapus tabel debug (membuat file lebih kecil dan sulit di-reverse engineer oleh hacker)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/server ./cmd/api/main.go

# Menggunakan image distroless varian "static" yang dikhususkan untuk binary Go.
# Tag ":nonroot" menyediakan user non-root secara bawaan.
FROM gcr.io/distroless/static-debian12:nonroot

# Copy hanya file binary sebelumnya, tinggalkan semua compiler dan source code
COPY --from=builder /app/server /server

# NON-ROOT CONTAINER
# Menjalankan aplikasi sebagai user biasa, bukan administrator (root)
USER nonroot:nonroot

# entrypoint eksekusi utama
ENTRYPOINT ["/server"]
