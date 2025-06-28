# GSalt Core - Payment Gateway & Loyalty System

GSalt Core adalah sistem internal Safatanc Group untuk payment gateway, manajemen loyalty points, balance management, dan voucher redemption dengan fitur topup, transfer, transaksi, dan gift.

## 🚀 Features

- **Account Management**: Manajemen akun pengguna dengan balance dan loyalty points
- **Transaction Processing**: Topup, transfer, payment, dan voucher redemption
- **Voucher System**: Manajemen voucher dengan berbagai tipe (balance, loyalty points, discount)
- **Authentication**: Integrasi dengan Safatanc Connect untuk autentikasi
- **Database Transactions**: Atomic operations untuk data integrity

## 🛠 Tech Stack

- **Backend**: Go (Fiber framework)
- **Database**: PostgreSQL dengan GORM
- **Authentication**: JWT via Safatanc Connect
- **Dependency Injection**: Google Wire
- **Validation**: go-playground/validator

## 📦 Installation

1. Clone repository
```bash
git clone https://github.com/safatanc/gsalt-core.git
cd gsalt-core
```

2. Install dependencies
```bash
go mod tidy
```

3. Setup environment variables
```bash
cp .env.example .env
# Edit .env file with your configuration
```

4. Run database migrations
```bash
# Run your migration files in migrations/ folder
```

5. Generate Wire dependencies
```bash
cd injector && wire
```

6. Run application
```bash
go run cmd/app/main.go
```

## 🌐 API Documentation

Base URL: `http://localhost:8080/api/v1`

### Authentication

Semua endpoint yang memerlukan autentikasi menggunakan header:
```
Authorization: Bearer <your-access-token>
```

### Health Check

#### GET /health
Check application health status.

**Response:**
```json
{
  "status": "ok",
  "service": "gsalt-core"
}
```

---

## 👤 Account Management

### GET /accounts/me
Get current user account information.

**Headers:** `Authorization: Bearer <token>`

**Response:**
```json
{
  "success": true,
  "data": {
    "connect_id": "uuid",
    "balance": 100000,
    "points": 500,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

### PATCH /accounts/me
Update current user account.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "balance": 150000,
  "points": 750
}
```

### DELETE /accounts/me
Delete current user account (soft delete).

**Headers:** `Authorization: Bearer <token>`

### GET /accounts/:id
Get account by connect ID.

**Response:** Same as GET /accounts/me

---

## 💰 Transaction Management

### POST /transactions
Create a new transaction.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "account_id": "uuid",
  "type": "topup|transfer_in|transfer_out|payment|voucher_redemption",
  "amount": "100000.00",
  "currency": "IDR",
  "description": "Transaction description",
  "source_account_id": "uuid", // optional
  "destination_account_id": "uuid", // optional
  "voucher_code": "VOUCHER123", // optional
  "external_reference_id": "ext-ref-123" // optional
}
```

### GET /transactions/me
Get current user's transactions with pagination.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**
- `limit` (default: 10)
- `offset` (default: 0)

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "account_id": "uuid",
      "type": "topup",
      "amount": "100000.00",
      "currency": "IDR",
      "status": "completed",
      "description": "Balance topup",
      "created_at": "2024-01-01T00:00:00Z",
      "completed_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### GET /transactions/:id
Get specific transaction by ID.

**Headers:** `Authorization: Bearer <token>`

### PATCH /transactions/:id
Update transaction.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "status": "completed|pending|failed",
  "description": "Updated description"
}
```

### POST /transactions/topup
Process balance topup.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "amount": "100000.00",
  "external_reference_id": "payment-gateway-ref-123"
}
```

### POST /transactions/transfer
Transfer balance between accounts.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "destination_account_id": "uuid",
  "amount": "50000.00",
  "description": "Transfer to friend"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "transfer_out": { /* transaction object */ },
    "transfer_in": { /* transaction object */ }
  }
}
```

### POST /transactions/payment
Process payment (deduct balance).

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "amount": "25000.00",
  "description": "Payment for service",
  "external_reference_id": "merchant-ref-123"
}
```

---

## 🎫 Voucher Management

### GET /vouchers
Get list of vouchers (public endpoint).

**Query Parameters:**
- `limit` (default: 10)
- `offset` (default: 0)
- `status` (active|expired|redeemed)

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "code": "WELCOME2024",
      "name": "Welcome Bonus",
      "description": "Welcome bonus for new users",
      "type": "balance|loyalty_points|discount",
      "value": "50000.00",
      "currency": "IDR",
      "loyalty_points_value": 100,
      "discount_percentage": 10.5,
      "discount_amount": "5000.00",
      "max_redeem_count": 1000,
      "current_redeem_count": 245,
      "valid_from": "2024-01-01T00:00:00Z",
      "valid_until": "2024-12-31T23:59:59Z",
      "status": "active",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### GET /vouchers/:id
Get voucher by ID (public endpoint).

### GET /vouchers/code/:code
Get voucher by code (public endpoint).

### POST /vouchers/validate/:code
Validate voucher eligibility (public endpoint).

**Response:**
```json
{
  "success": true,
  "data": {
    "valid": true,
    "voucher": { /* voucher object */ }
  }
}
```

### POST /vouchers
Create new voucher (protected endpoint).

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "code": "NEWVOUCHER2024",
  "name": "New Year Voucher",
  "description": "Special voucher for new year",
  "type": "balance",
  "value": "100000.00",
  "currency": "IDR",
  "max_redeem_count": 500,
  "valid_from": "2024-01-01T00:00:00Z",
  "valid_until": "2024-01-31T23:59:59Z"
}
```

### PATCH /vouchers/:id
Update voucher (protected endpoint).

**Headers:** `Authorization: Bearer <token>`

### DELETE /vouchers/:id
Delete voucher (protected endpoint).

**Headers:** `Authorization: Bearer <token>`

---

## 🎁 Voucher Redemption

### POST /voucher-redemptions/redeem
Redeem voucher (main endpoint).

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "voucher_code": "WELCOME2024"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "redemption": {
      "id": "uuid",
      "voucher_id": "uuid",
      "account_id": "uuid",
      "transaction_id": "uuid",
      "redeemed_at": "2024-01-01T00:00:00Z"
    },
    "transaction": {
      "id": "uuid",
      "type": "voucher_redemption",
      "amount": "50000.00",
      "status": "completed"
    }
  }
}
```

### GET /voucher-redemptions/me
Get current user's redemption history.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**
- `limit` (default: 10)
- `offset` (default: 0)

### GET /voucher-redemptions/:id
Get specific redemption by ID.

**Headers:** `Authorization: Bearer <token>`

### GET /voucher-redemptions/voucher/:voucher_id
Get redemptions by voucher ID (admin endpoint).

**Headers:** `Authorization: Bearer <token>`

### POST /voucher-redemptions
Create redemption manually (admin endpoint).

**Headers:** `Authorization: Bearer <token>`

### PATCH /voucher-redemptions/:id
Update redemption.

**Headers:** `Authorization: Bearer <token>`

### DELETE /voucher-redemptions/:id
Delete redemption.

**Headers:** `Authorization: Bearer <token>`

---

## 📊 Voucher Types

### 1. Balance Voucher (`balance`)
- Menambahkan saldo ke akun pengguna
- Field yang diperlukan: `value`, `currency`
- Contoh: Voucher saldo Rp 50.000

### 2. Loyalty Points Voucher (`loyalty_points`)
- Menambahkan loyalty points ke akun pengguna
- Field yang diperlukan: `loyalty_points_value`
- Contoh: Voucher 100 loyalty points

### 3. Discount Voucher (`discount`)
- Memberikan diskon untuk pembayaran
- Field yang diperlukan: `discount_percentage` atau `discount_amount`
- Contoh: Diskon 10% atau potongan Rp 5.000

---

## 🔧 Environment Variables

```env
# Database
DATABASE_URL=postgresql://user:password@localhost:5432/gsalt_core

# Server
PORT=8080

# Safatanc Connect (for authentication)
CONNECT_BASE_URL=https://connect.safatanc.com
CONNECT_CLIENT_ID=your-client-id
CONNECT_CLIENT_SECRET=your-client-secret
```

---

## 🚦 Response Format

### Success Response
```json
{
  "success": true,
  "data": { /* response data */ }
}
```

### Error Response
```json
{
  "success": false,
  "message": "Error description"
}
```

---

## 📝 Transaction Status

- `pending`: Transaksi sedang diproses
- `completed`: Transaksi berhasil diselesaikan
- `failed`: Transaksi gagal

## 🎫 Voucher Status

- `active`: Voucher aktif dan dapat digunakan
- `expired`: Voucher sudah kedaluwarsa
- `redeemed`: Voucher sudah mencapai batas maksimum redemption

---

## 🔒 Security

- Semua endpoint protected menggunakan JWT authentication via Safatanc Connect
- Input validation menggunakan go-playground/validator
- Database transactions untuk memastikan data integrity
- Soft delete untuk data audit trail

---

## 📞 Support

Untuk pertanyaan dan dukungan, hubungi tim development Safatanc Group.

**Built with ❤️ by Safatanc Group**
