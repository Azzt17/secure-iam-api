# 🛡️ Secure IAM & Wallet API (DevSecOps Architecture)

**Author:** Farid Wajdi  
**Tech Stack:** Go (Golang) Standard Library (`net/http`), PostgreSQL, OpenSSL, `golang-migrate`

---

Sebuah _technical laboratory_ dan _Proof of Concept_ (PoC) untuk membangun sistem **Identity & Access Management (IAM)** dan **Transaksi Dompet Digital** yang sepenuhnya berorientasi pada keamanan (_Secure-by-Design_).

Proyek ini sengaja dibangun **tanpa menggunakan web framework** (seperti Gin atau Fiber) dan **tanpa ORM** untuk mendemonstrasikan penguasaan fundamental terhadap protokol HTTP internal Go, manajemen memori, eksekusi _raw SQL_, serta mitigasi kerentanan keamanan di berbagai lapisan arsitektur (Layer 4 hingga Layer Data).

---

## 🏗️ Arsitektur Pertahanan Berlapis (Defense-in-Depth)

Sistem ini menerapkan prinsip _Zero Trust_ dan diarsiteki dengan 7 lapisan pertahanan utama:

1. **ACID Transactions & Row-Level Locking (Data Layer)**  
   Menghindari _Time-of-Check to Time-of-Use_ (TOCTOU) dan _Race Conditions_ finansial menggunakan pola transaksi `sql.Tx` dengan jaring pengaman `defer tx.Rollback()`. Mutasi saldo dikunci secara presisi menggunakan `SELECT ... FOR UPDATE` di tingkat PostgreSQL.

2. **Persistence Security & Connection Pool (Database Layer)**  
   Menerapkan _Parameterized Queries_ (`$1`, `$2`) untuk menetralisir _SQL Injection_ (SQLi) secara mutlak. Koneksi dikelola secara ketat melalui pembatasan _Connection Pool_ (`SetMaxOpenConns`, `SetMaxIdleConns`) untuk mencegah _Resource Exhaustion_ (DDoS memori server DB).

3. **Whitelist Input Sanitization (Application Layer)**  
   Menggunakan `go-playground/validator` untuk memaksa batas panjang input dan aturan _alphanum_, menyaring vektor _Cross-Site Scripting_ (XSS) dan injeksi data kotor sebelum menyentuh logika bisnis.

4. **Secure Session Management (Session Layer)**  
   Menerbitkan JSON Web Token (JWT) dengan mitigasi kerentanan `alg: none`. Sesi dikirim secara eksklusif melalui perisai _Cookie_ berbendera `HttpOnly`, `Secure`, dan `SameSite=Strict` (mencegah XSS & CSRF).

5. **Gatekeeper & RBAC (Authorization Layer)**  
   Memanfaatkan _Context Values_ dalam _Middleware Chain_ untuk mengekstrak identitas JWT secara aman, memastikan bahwa operasi rahasia seperti `/wallet/deduct` hanya dieksekusi berdasarkan identitas kriptografis, bukan klaim JSON klien.

6. **Perimeter Firewall (Gateway Layer)**  
   Injeksi _Middleware Decorators_ untuk menolak serangan secara preemptif:
   - **Rate Limiter:** Mencegah serangan _Brute-force_ dan volumetrik menggunakan pelacakan IP `net.SplitHostPort`.
   - **Security Headers:** Memaksa HSTS, mencegah _MIME Sniffing_ (`nosniff`), dan _Clickjacking_ (`DENY`).
   - **Panic Recovery:** Menangkap _runtime errors_ untuk mencegah kebocoran _Stack Trace_ memori OS ke publik.

7. **Encrypted Tunnel (Transport Layer)**  
   Konfigurasi `tls.Config` kustom yang secara eksplisit menonaktifkan protokol usang (TLS 1.0/1.1) dan memaksa koneksi HTTPS menggunakan mekanisme kriptografi modern.

8. **Layered Architecture & Dependency Injection (Clean Code)**
   Sistem dipisahkan secara ketat menjadi 3 lapisan (Handler, Service, Repository). Pemisahan tanggung jawab (Separation of Concerns) ini dicapai melalui injeksi antarmuka (Interface Injection), membuat logika bisnis terisolasi dari protokol HTTP dan kueri SQL.

---

## 📂 Struktur Repositori Standar Industri

```text
.
├── certs/                      # Sertifikat TLS (Diabaikan di .git)
├── cmd/
│   └── api/
│       └── main.go             # Entry point, Dependency Injection (DI) Container, & Router
├── internal/
│   ├── auth/                   # Engine Kriptografi: Bcrypt Hashing & JWT
│   ├── db/                     # Konfigurasi PostgreSQL Connection Pool
│   ├── domain/                 # Entitas Murni & Definisi Error Sentral (Tanpa dependensi eksternal)
│   ├── handler/                # Layer 1: HTTP Resepsionis (Parsing JSON & Response Formatting)
│   ├── middleware/             # Layer 7 Firewalls (RateLimit, CORS, Headers, Gatekeeper)
│   ├── repository/             # Layer 3: Interaksi Database Raw SQL (PostgreSQL & Mocking)
│   └── service/                # Layer 2: Pusat Logika Bisnis & Pengujian Terisolasi
├── migrations/                 # Skema Database (Infrastructure as Code / golang-migrate)
├── .env                        # Kredensial Database & Secret (Diabaikan di .git)
├── go.mod
├── go.sum
└── README.md
```

---

## 🚀 Panduan Eksekusi Lokal

### 1. Prasyarat Sistem

- Go `1.20+`
- PostgreSQL `15+`
- Utilitas `openssl`
- CLI `golang-migrate`

### 2. Konfigurasi Database (Least Privilege)

Masuk ke PostgreSQL dan siapkan basis data beserta user dengan akses terbatas:

```sql
CREATE DATABASE iam_wallet_db;
CREATE USER iam_app WITH PASSWORD '<your_dbpassword>';
GRANT ALL PRIVILEGES ON DATABASE iam_wallet_db TO iam_app;
-- Masuk ke database iam_wallet_db lalu jalankan:
GRANT ALL ON SCHEMA public TO iam_app;
```

### 3. Migrasi Skema (Infrastructure as Code)

Jalankan perintah ini untuk membangun tabel `users` dan `wallets` secara otomatis:

```bash
migrate -path migrations -database "postgres://iam_app:<your_dbpassword>@localhost:5432/iam_wallet_db?sslmode=disable" up
```

### 4. Variabel Lingkungan (`.env`)

Buat file `.env` di direktori root proyek dengan konfigurasi berikut:

```env
DB_HOST=<your_db_url>
DB_PORT=5432
DB_USER=iam_app
DB_PASSWORD=<your_dbpassword>
DB_NAME=iam_wallet_db
JWT_SECRET=<your_jwt_secret>
DB_SSLMODE=disable/enable
FRONTEND_URL=<your_frontend_url>
```

### 5. Membangun Kriptografi Transport (TLS)

Buat Self-Signed Certificate lokal di dalam folder `certs`:

```bash
mkdir -p certs
openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt -days 365 -nodes -subj "/CN=localhost"
```

### 6. Menjalankan Server

```bash
go run cmd/api/main.go
# Output:
#  Koneksi PostgreSQL (Connection Pool) berhasil diamankan!
#  Secure IAM API berjalan di port :8443...
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

Perintah ini menggunakan `-b cookies.txt` untuk mengirim JWT ke brankas sistem dan menguji mekanisme Row-Level Locking PostgreSQL.

```bash
curl -k -b cookies.txt -X POST https://localhost:8443/wallet/deduct
```

### 4. Menguji Perimeter Keamanan (Rate Limiter DoS Test)

Berondong server dengan cepat. Pada request ke-6, Anda akan diblokir dengan HTTP `429 Too Many Requests`.

```bash
for i in {1..6}; do curl -k -i -X GET https://localhost:8443/health; echo ""; done
```

### 5. Pengujian Terisolasi (Unit Testing & Mocking)

Sistem ini menggunakan teknik _Functional Mocking_ pada lapisan _Repository_ untuk menguji logika bisnis di _Service Layer_ secara terisolasi, tanpa memerlukan koneksi ke database nyata.

Jalankan pengujian untuk memverifikasi jalur sukses dan skenario _error_ (seperti saldo tidak mencukupi):

```bash
go test -v ./internal/service
# Output: Skenario sukses dan gagal tervalidasi dalam hitungan milidetik (< 0.01s)
```

---

_Dikembangkan sebagai bagian dari eksplorasi mendalam terhadap arsitektur backend, manajemen memori persisten, dan standar keamanan siber OWASP._
