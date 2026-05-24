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
│   ├── handlers/                # Layer 1: HTTP Resepsionis (Parsing JSON & Response Formatting)
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

- Go `1.21+`
- PostgreSQL `15+`
- Utilitas `openssl`
- CLI `golang-migrate`

### 2. Konfigurasi Database (Least Privilege)

Masuk ke PostgreSQL dan siapkan basis data beserta user dengan akses terbatas:
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImZhcmlkd2FqZGkiLCJyb2xlIjoidXNlciIsImV4cCI6MTc3OTYyMzA0MSwiaWF0IjoxNzc5NjE5NDQxfQ.Q6VEOuol8bMpXgdsTLH9J3Eh9Cl7-wZa8yo9VteJit4

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

### 6. Analisis Keamanan Statis & Orkestrasi Linter (SAST - DevSecOps Suite)

Untuk menjamin kode sepenuhnya patuh terhadap standar keamanan OWASP, bersih dari _anti-pattern_, dan efisien dalam manajemen memori, repositori ini mengintegrasikan **`golangci-lint` (Skema Versi 2)** sebagai agregator dan konduktor otomatis untuk menjalankan rangkaian inspeksi statis berikut secara paralel:

- **govet:** Memeriksa anomali logika dasar bawaan Go dan akurasi tanda tangan fungsi.
- **staticcheck:** Mengaudit efisiensi kode, arsitektur performa, dan mendeteksi API usang (_deprecated_).
- **gosec:** Memindai _Abstract Syntax Tree_ (AST) untuk memburu celah keamanan kritis (Injeksi SQL, kebocoran kredensial, kelemahan kriptografi, dsb).
- **errcheck:** Memastikan akuntabilitas penanganan _error_, memaksa kepatuhan deteksi kegagalan pada operasi _defer_ seperti `db.Close()` dan `tx.Rollback()`.
- **ineffassign:** Mendeteksi inefisiensi alokasi variabel yang tidak pernah digunakan.
- **bodyclose:** Mencegah risiko kebocoran memori (_resource leak_) dengan memastikan seluruh HTTP tanggapan selalu ditutup dengan benar.

Seluruh aturan main ini dikonfigurasi secara terpusat di dalam berkas tata tertib `.golangci.yml`.

Jalankan orkestrasi analisis statis tunggal ini di direktori _root_:

```bash
golangci-lint run ./...
```

Metrik Keberhasilan Audit Terakhir:

```plaintext
0 issues. (Sistem dinyatakan SEPENUHNYA PAS DAN LOLOS AUDIT)
```

### 7. Analisis Rantai Pasok Perangkat Lunak (SCA - Supply Chain Security)

Repositori ini menerapkan keamanan Rantai Pasok lapis ganda untuk mencegah masuknya kerentanan dari pustaka eksternal dan memitigasi serangan rekayasa sosial seperti _Typosquatting_:

- **Integritas Kriptografi (`go.sum`):** Memvalidasi _hash_ SHA-256 dari setiap dependensi terhadap _database checksum_ global Google (`sum.golang.org`) guna menjamin perlindungan dari manipulasi modul di tengah jalan (_man-in-the-middle_).
- **Pemindaian Kerentanan Aktif (`govulncheck`):** Menggunakan alat analisis resmi dari Golang dengan pendekatan _Call Graph Analysis_ untuk melacak _Common Vulnerabilities and Exposures_ (CVE). Alat ini secara spesifik hanya akan menyorot fungsi rentan yang benar-benar dieksekusi oleh aplikasi, memastikan nol _false positives_.

Jalankan audit rantai pasok secara berkala dengan perintah:

```bash
govulncheck ./...
```

Status Rantai Pasok Terakhir: 0 vulnerabilities affecting the code (Aman / Tervalidasi).

### 8. Keamanan Kontainer (Container Security)

Aplikasi ini dikemas menggunakan standar arsitektur _DevSecOps_ mutakhir untuk memastikan isolasi absolut dan meminimalisir _Attack Surface_ di lingkungan _Production_:

- **Multi-Stage Builds:** Memisahkan lingkungan kompilasi dan lingkungan _runtime_. Biner dikompilasi secara statis (`CGO_ENABLED=0`) dan jejak alat _build_ (kompilator, kode sumber) dimusnahkan dari rilis final.
- **Distroless Base Image:** Menggunakan `gcr.io/distroless/static-debian12`. Tidak ada _shell_ (`bash`/`sh`), tidak ada utilitas jaringan (`curl`/`wget`), dan tidak ada _package manager_. Memblokir penuh taktik pergerakan lateral peretas.
- **Non-Root & Read-Only:** _Container_ dieksekusi secara ketat sebagai pengguna `nonroot` dengan hak akses terbatas. Disarankan untuk menjalankan _container_ dengan bendera `--read-only` untuk menciptakan _immutable filesystem_ yang mencegah injeksi _script_ berbahaya (RCE).
- **Inspeksi Artefak (Trivy):** _Image_ kontainer diaudit secara rutin menggunakan Aqua Security Trivy untuk memindai kerentanan OS (_Debian_) dan ketergantungan _binary_.

_Metrik Keamanan Kontainer Terakhir:_ Ukuran _Image_ 20.4MB | `0 vulnerabilities` terdeteksi pada _base image_ dan _gobinary_.

### 9. Panduan Menjalankan Kontainer (Konteks Lokal)

Karena kontainer ini dibangun dengan isolasi mutlak (_distroless_, _read-only_, dan _non-root_), ia tidak menyimpan kredensial atau sertifikat di dalam _image_. Semua konfigurasi harus disuntikkan secara dinamis saat _runtime_ (saat kontainer dihidupkan).

Gunakan perintah di bawah ini untuk menjalankan server API secara lokal dengan aman, memastikan Anda mengeksekusinya tepat di **root direktori proyek**:

```bash
docker run --rm -it \
  --name secure-iam-api \
  --read-only \
  --network host \
  -w /app \
  -v "$(pwd)/certs:/app/certs:ro,z" \
  -e DB_HOST=localhost \
  -e DB_PORT=5432 \
  -e DB_USER=iam_app \
  -e DB_PASSWORD=<your_dbpassword> \
  -e DB_NAME=iam_wallet_db \
  -e DB_SSLMODE=require/disable \
  -e JWT_SECRET="<your_jwt_secret>" \
  secure-iam-api:v1
```

Anatomi Perintah Keamanan:
--read-only: Mengunci filesystem kontainer agar tidak dapat ditulisi (mencegah modifikasi malware).
--network host: Membuka akses jaringan agar kontainer dapat berkomunikasi dengan PostgreSQL di mesin lokal.
-w /app: Menetapkan direktori kerja yang spesifik untuk akurasi pencarian path sertifikat.
-v ... :ro,z: Melakukan mounting sertifikat TLS secara Read-Only (ro) dan menyuntikkan label izin khusus SELinux (z) agar kontainer non-root diizinkan membaca file dari OS Host.

### 10. CI/CD Security Pipeline (Otomatisasi DevSecOps)

Proyek ini dilengkapi dengan ban berjalan terotomatisasi (_pipeline_) menggunakan **GitHub Actions** (`.github/workflows/devsecops.yml`). Pipeline ini dirancang dengan filosofi _Fail Fast, Fail Cheap_ untuk menolak kode yang tidak aman sedini mungkin sebelum mencapai tahap _build_.

Alur inspeksi keamanan berjalan secara berurutan:

1. **Secret Scanning (Trivy FS):** Memblokir _pipeline_ jika terdapat _hardcoded credentials_ (seperti API Key atau sandi) di dalam kode sumber.
2. **Linting & Code Quality:** Menggunakan `go vet` dan `staticcheck` untuk memastikan kode bebas dari kejanggalan sintaks dan memenuhi standar idiomatis Go yang tangguh.
3. **SAST (GoSec):** Menganalisis kelemahan logika keamanan pada kode (misal: _SQL Injection_, kelemahan kriptografi) tanpa harus mengeksekusi aplikasi.
4. **SCA (Govulncheck):** Memeriksa seluruh hierarki pustaka pihak ketiga (_dependencies_) terhadap _database_ kerentanan (_CVE_) nasional.
5. **Container Security (Trivy Image):** Setelah _image_ Docker berbasis _distroless_ selesai dibangun, Trivy kembali memindai hasil akhir _image_ untuk memastikan tidak ada celah di level OS atau _binary_.
6. **Supply Chain Transparency (Syft):** _Pipeline_ secara otomatis mengekstrak **SBOM (Software Bill of Materials)** dalam format SPDX JSON. Ini bertindak sebagai Label yang mendata seluruh komponen di dalam _image_, sangat krusial untuk _Incident Response_ jika terjadi serangan _Zero-Day_ di masa depan.

### 11. Secure Observability & Audit Logging

Sistem ini membuang pencatatan log teks datar (_plaintext_) standar dan mengimplementasikan **Structured JSON Logging** menggunakan pustaka bawaan `log/slog`. Pendekatan ini dirancang untuk memfasilitasi pemantauan terpusat (seperti ELK Stack atau Datadog) sekaligus memitigasi kerentanan keamanan level pemantauan:

1. **Pencegahan Log Injection (CRLF Attack):**
   Masukan nakal dari pengguna yang berisi karakter _newline_ (`\n` atau `\r`) secara otomatis di-_escape_ oleh `JSONHandler`. Peretas tidak dapat memanipulasi log _parser_ untuk menyuntikkan baris log palsu.
2. **Sanitasi PII (Personally Identifiable Information) Otomatis:**
   Sistem diinjeksi dengan _middleware_ internal `ReplaceAttr` pada level _logger_. Setiap _key_ yang terdeteksi sebagai data sensitif (seperti `password`, `token`, atau `secret`) akan secara mutlak disensor menjadi `[REDACTED]`. Alamat `email` juga secara dinamis disamarkan (contoh: `f***d@example.com`) sebelum dicetak ke _stdout_, mencegah insiden kebocoran data (_Data Leak_) ke dasbor pemantauan.
3. **Traceability & Status Interception:**
   Setiap HTTP _request_ yang masuk dibekali dengan `request_id` unik untuk keperluan _Distributed Tracing_. Karena antarmuka standar Go tidak menyimpan _HTTP Status Code_, sistem menggunakan pola `responseRecorder` untuk menyadap dan mencatat `status_code` serta `duration_ms` di lapisan _middleware_ paling dalam. Ini memungkinkan visibilitas penuh terhadap serangan volumetrik (seperti _Rate Limit / 429 Too Many Requests_).

### 12. Real-time Monitoring Infrastructure (Prometheus & Grafana)

Repositori ini menyertakan infrastruktur pemantauan instan berbasis Docker Compose untuk memvisualisasikan metrik keamanan dan performa aplikasi secara _real-time_.

**Cara Menjalankan Dasbor Pemantauan:**

1. Pastikan API Go sedang berjalan (mendengarkan di port `:8443`).
2. Masuk ke direktori `observability` dan jalankan orkestrasi kontainer:

   ```bash
   cd observability
   docker compose up -d
   ```

3. Buka Grafana di <http://localhost:3000> (Kredensial bawaan: admin / admin).
4. Grafana telah dikonfigurasi untuk secara otomatis menghisap data dari metrik Prometheus yang diekspos secara aman oleh aplikasi Go. Anda dapat memantau anomali keamanan (seperti lonjakan HTTP 401 dan 429) serta memprofilkan kesehatan goroutine melalui antarmuka visual.

---

_Dikembangkan sebagai bagian dari eksplorasi mendalam terhadap arsitektur backend, manajemen memori persisten, dan standar keamanan siber OWASP._
