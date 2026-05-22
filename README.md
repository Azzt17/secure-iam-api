# 🛡️ Secure IAM & Wallet API (DevSecOps Architecture)

**Author:** Farid Wajdi  
**Tech Stack:** Go (Golang) Standard Library (`net/http`), OpenSSL

---

Sebuah _technical laboratory_ dan _Proof of Concept_ (PoC) untuk membangun sistem **Identity & Access Management (IAM)** dan **Transaksi Dompet Digital** yang sepenuhnya berorientasi pada keamanan (_Secure-by-Design_).

Proyek ini sengaja dibangun **tanpa menggunakan web framework** (seperti Gin atau Fiber) untuk mendemonstrasikan penguasaan fundamental terhadap protokol HTTP internal Go, siklus hidup _goroutine_, dan mitigasi kerentanan keamanan di berbagai lapisan OSI (Layer 4 hingga Layer 7).

---

## 🏗️ Arsitektur Pertahanan Berlapis (Defense-in-Depth)

Sistem ini menerapkan prinsip _Zero Trust_ dan diarsiteki dengan 6 lapisan pertahanan utama:

1. **Memory Synchronization (Data Layer)**  
   Menerapkan `sync.RWMutex` pada basis data in-memory dompet pengguna untuk mencegah eksploitasi _Time-of-Check to Time-of-Use_ (TOCTOU) dan _Race Conditions_ pada pemrosesan transaksi paralel.

2. **Whitelist Input Sanitization (Application Layer)**  
   Menggunakan `go-playground/validator` untuk memaksa batas panjang input dan aturan ketat `alphanum`, menetralisir vektor serangan _SQL Injection_ (SQLi) dan _Cross-Site Scripting_ (XSS) sebelum menyentuh logika bisnis.

3. **Secure Session Management (Session Layer)**  
   Menerbitkan JSON Web Token (JWT) dengan mitigasi eksploitasi `alg: none`, dan mengirimkannya secara eksklusif melalui perisai `HttpOnly`, `Secure`, dan `SameSite=Strict` cookies (mencegah XSS & CSRF).

4. **Gatekeeper & RBAC (Authorization Layer)**  
   Memanfaatkan _Context Values_ dalam _Middleware Chain_ untuk mengekstrak identitas JWT secara aman dan melindungi rute privat (seperti `/wallet/deduct`).

5. **Perimeter Firewall (Gateway Layer)**  
   Injeksi _Middleware Decorators_ untuk menolak serangan secara preemptif:
   - **Rate Limiter:** Mencegah serangan _Brute-force_ dan volumetrik (DDoS) menggunakan pelacakan IP `net.SplitHostPort`.
   - **Security Headers:** Memaksa HSTS, mencegah _MIME Sniffing_ (`nosniff`), dan _Clickjacking_ (`DENY`).
   - **Panic Recovery:** Menangkap _runtime errors_ untuk mencegah kebocoran _Stack Trace_ memori OS ke publik.

6. **Encrypted Tunnel (Transport Layer)**  
   Konfigurasi `tls.Config` kustom yang secara eksplisit menonaktifkan protokol usang (TLS 1.0/1.1) dan memaksa koneksi HTTPS menggunakan Cipher Suites modern.

---

## 📂 Struktur Repositori Standar Industri

```text
.
├── certs/                      # Sertifikat TLS (server.key - diabaikan di .git)
├── cmd/
│   └── api/
│       └── main.go             # Entry point, TLS Config, dan Registrasi Router (ServeMux)
├── internal/
│   ├── auth/                   # Engine Kriptografi: Bcrypt Hashing & JWT Generation/Validation
│   └── middleware/             # Layer 7 Firewalls (RateLimit, CORS, Headers, Auth Gatekeeper)
├── go.mod
├── go.sum
└── README.md
```

---

## 🚀 Panduan Eksekusi Lokal

### 1. Prasyarat Sistem

- Go `1.20+`
- Utilitas `openssl` (Bawaan pada distribusi Linux/Fedora)

### 2. Membangun Kriptografi Transport (TLS)

Karena proyek ini memaksa penggunaan HTTPS, Anda harus membuat Self-Signed Certificate lokal terlebih dahulu:

```bash
mkdir -p certs
openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt -days 365 -nodes -subj "/CN=localhost"
```

### 3. Menjalankan Server

```bash
go run cmd/api/main.go
# Output: Secure IAM API berjalan di port :8443...
```

---

## 🧪 Skenario Pengujian (Terminal / cURL)

> Karena menggunakan Self-Signed Certificate, gunakan flag `-k` atau `--insecure` pada cURL.

### 1. Mendaftarkan Entitas Baru (Registrasi)

```bash
curl -k -X POST https://localhost:8443/register \
     -H "Content-Type: application/json" \
     -d '{"username": "farid", "password": "rahasia123"}'
```

### 2. Autentikasi dan Mengambil Tiket Sesi (Login)

Perintah ini menggunakan `-c cookies.txt` untuk menyimpan Secure Cookie (JWT) secara otomatis.

```bash
curl -k -c cookies.txt -X POST https://localhost:8443/login \
     -H "Content-Type: application/json" \
     -d '{"username": "farid", "password": "rahasia123"}'
```

### 3. Mengakses Endpoint Terkunci (Potong Saldo)

Perintah ini menggunakan `-b cookies.txt` untuk mengirim JWT dari brankas Cookie.

```bash
curl -k -b cookies.txt -X POST https://localhost:8443/wallet/deduct
```

### 4. Menguji Perimeter Keamanan (Rate Limiter DoS Test)

Berondong server dengan cepat. Pada request ke-6, Anda akan diblokir dengan HTTP `429 Too Many Requests`.

```bash
for i in {1..6}; do curl -k -i -X GET https://localhost:8443/health; echo ""; done
```

---

_Dikembangkan sebagai bagian dari eksplorasi mendalam terhadap arsitektur backend, manajemen memori Go, dan standar keamanan siber OWASP._
