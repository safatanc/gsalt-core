# GSalt Core - Payment Gateway & Digital Wallet System

GSalt Core is Safatanc Group's payment gateway and digital wallet system that supports topup, transfer, payment, and voucher redemption with GSALT (Safatanc Global Loyalty Token) system. This system functions as a multi-currency payment gateway and digital wallet with automatic conversion to GSALT units.

## Features

- **Digital Wallet**: Balance management in GSALT units with multi-currency conversion
- **Payment Gateway**: Payment processing through various methods (QRIS, Bank Transfer, GSALT Balance)
- **Multi-Currency Support**: Supports IDR, USD, EUR, SGD with automatic exchange rates
- **Transaction Processing**: Topup, transfer, payment with atomic operations
- **Voucher System**: Voucher management with various types and GSALT conversion
- **GSALT Exchange**: 1 GSALT = 100 units (2 decimal places), default 1000 IDR = 1 GSALT

## Tech Stack

- **Backend**: Go (Fiber framework)
- **Database**: PostgreSQL with GORM
- **Authentication**: JWT via Safatanc Connect
- **Dependency Injection**: Google Wire
- **Validation**: go-playground/validator

## GSALT System

### GSALT Units
- **1 GSALT = 100 units** (for 2 decimal precision)
- **Default Exchange Rate**: 1000 IDR = 1 GSALT
- **Balance Storage**: Stored in GSALT units (int64)
- **Supported Currencies**: IDR, USD, EUR, SGD

### Payment Methods
- `GSALT_BALANCE`: Pay using GSALT balance
- `QRIS`: Pay via QRIS
- `BANK_TRANSFER`: Bank transfer
- `CREDIT_CARD`: Credit card
- `DEBIT_CARD`: Debit card

## Installation

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

## API Documentation

Base URL: `http://localhost:8080`

### Authentication

All endpoints requiring authentication use header:
```
Authorization: Bearer <your-access-token>
```

### Health Check

#### GET /health
Check application health status.

**Response:**
```json
{
  "success": true,
  "data": "gsalt-core"
}
```

---

## Account Management

### GET /accounts/me
Get current user account information.

**Headers:** `Authorization: Bearer <token>`

**Response:**
```json
{
  "success": true,
  "data": {
    "connect_id": "uuid",
    "balance": 1000000,
    "points": 500,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

**Notes:**
- Balance is in GSALT units (1000000 units = 10,000 GSALT)

### GET /accounts/:id
Get account by connect ID (public endpoint).

**Parameters:**
- `id` (string): Account connect ID

**Response:** Same format as GET /accounts/me

### DELETE /accounts/me
Delete current user account (soft delete).

**Headers:** `Authorization: Bearer <token>`

**Response:**
```json
{
  "success": true,
  "data": null
}
```

---

## Transaction Management

### POST /transactions
Create a new transaction manually.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "account_id": "uuid",
  "type": "topup|transfer_in|transfer_out|payment|voucher_redemption|gift_in|gift_out",
  "amount_gsalt_units": 100000,
  "currency": "GSALT",
  "exchange_rate_idr": "1000.00",
  "payment_amount": 1000000,
  "payment_currency": "IDR",
  "payment_method": "QRIS",
  "description": "Transaction description",
  "source_account_id": "uuid",
  "destination_account_id": "uuid",
  "voucher_code": "VOUCHER123",
  "external_reference_id": "ext-ref-123"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "account_id": "uuid",
    "type": "topup",
    "amount_gsalt_units": 100000,
    "currency": "GSALT",
    "exchange_rate_idr": "1000.00",
    "payment_amount": 1000000,
    "payment_currency": "IDR",
    "payment_method": "QRIS",
    "status": "pending",
    "description": "Transaction description",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### GET /transactions/me
Get current user's transactions with pagination.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**
- `page` (default: 1): Page number to return
- `limit` (default: 10): Number of transactions per page
- `order` (default: desc): Sort order (asc|desc)
- `order_field` (default: created_at): Field to sort by

**Response:**
```json
{
  "success": true,
  "data": {
    "page": 1,
    "limit": 10,
    "total_pages": 5,
    "total_items": 42,
    "has_next": true,
    "has_prev": false,
    "items": [
      {
        "id": "uuid",
        "account_id": "uuid",
        "type": "topup",
        "amount_gsalt_units": 100000,
        "currency": "GSALT",
        "exchange_rate_idr": "1000.00",
        "payment_amount": 1000000,
        "payment_currency": "IDR",
        "payment_method": "QRIS",
        "status": "completed",
        "description": "Balance topup",
        "created_at": "2024-01-01T00:00:00Z",
        "completed_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

### GET /transactions/:id
Get specific transaction by ID.

**Headers:** `Authorization: Bearer <token>`

**Parameters:**
- `id` (string): Transaction ID

**Response:** Same format as single transaction object

### PATCH /transactions/:id
Update transaction status and details.

**Headers:** `Authorization: Bearer <token>`

**Parameters:**
- `id` (string): Transaction ID

**Request Body:**
```json
{
  "status": "completed|pending|failed|cancelled",
  "exchange_rate_idr": "1000.00",
  "payment_amount": 1000000,
  "payment_currency": "IDR",
  "payment_method": "QRIS",
  "description": "Updated description"
}
```

**Response:** Updated transaction object

### POST /transactions/topup
Process balance topup via external payment.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "amount_gsalt": "100.00",
  "payment_amount": 100000,
  "payment_currency": "IDR",
  "payment_method": "QRIS",
  "external_reference_id": "payment-gateway-ref-123"
}
```

**Notes:**
- `amount_gsalt`: Amount in GSALT (will be converted to units)
- `payment_amount`: Actual payment amount (optional, defaults to exchange rate calculation)
- `payment_currency`: Payment currency (optional, defaults to IDR)
- `payment_method`: Payment method (optional, defaults to QRIS)

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "type": "topup",
    "amount_gsalt_units": 10000,
    "payment_amount": 100000,
    "payment_currency": "IDR",
    "payment_method": "QRIS",
    "status": "completed"
  }
}
```

### POST /transactions/transfer
Transfer GSALT balance between accounts.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "destination_account_id": "uuid",
  "amount_gsalt": "50.00",
  "description": "Transfer to friend"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "transfer_out": {
      "id": "uuid",
      "type": "transfer_out",
      "amount_gsalt_units": 5000,
      "currency": "GSALT",
      "status": "completed"
    },
    "transfer_in": {
      "id": "uuid",
      "type": "transfer_in", 
      "amount_gsalt_units": 5000,
      "currency": "GSALT",
      "status": "completed"
    }
  }
}
```

### POST /transactions/payment
Process payment (can use GSALT balance or external payment).

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "amount_gsalt": "25.00",
  "payment_amount": 25000,
  "payment_currency": "IDR",
  "payment_method": "GSALT_BALANCE",
  "description": "Payment for service",
  "external_reference_id": "merchant-ref-123"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "type": "payment",
    "amount_gsalt_units": 2500,
    "payment_method": "GSALT_BALANCE",
    "status": "completed"
  }
}
```

---

## Authentication & Middleware

GSalt Core uses a two-layer authentication system through Safatanc Connect integration with middleware validation.

### Authentication Flow

1. **Connect Authentication** (`AuthConnect`): Validates Safatanc Connect token
2. **Account Authentication** (`AuthAccount`): Validates GSALT account registration

### Middleware Components

#### AuthConnect Middleware
Validates the user's Safatanc Connect authentication token.

**Function**: Extracts and validates the Bearer token from Authorization header.

**Process**:
1. Extracts `Authorization` header
2. Removes "Bearer " prefix from token
3. Validates token with Safatanc Connect service
4. Sets `connect_user` in fiber context locals
5. Returns 401 if token is missing or invalid

**Usage**:
```go
// Applied to endpoints requiring Safatanc Connect authentication
router.Get("/protected", authMiddleware.AuthConnect, handler)
```

**Response on Failure**:
```json
{
  "success": false,
  "message": "Unauthorized"
}
```

#### AuthAccount Middleware
Validates that the authenticated Connect user has a registered GSALT account.

**Function**: Ensures Connect user is registered in GSALT system.

**Process**:
1. Retrieves `connect_user` from context (set by AuthConnect)
2. Fetches corresponding GSALT account using Connect user ID
3. Sets `account` in fiber context locals
4. Returns 401 if account not found

**Dependencies**: Must be used after `AuthConnect` middleware.

**Usage**:
```go
// Applied to endpoints requiring both Connect auth AND GSALT account
router.Get("/wallet", authMiddleware.AuthConnect, authMiddleware.AuthAccount, handler)
```

**Response on Failure**:
```json
{
  "success": false,
  "message": "User with connect username {username} is not registered on GSALT. Please register first."
}
```

### Context Locals

After successful authentication, the following objects are available in fiber context:

#### connect_user
Available after `AuthConnect` middleware.
```go
connectUser := c.Locals("connect_user").(*models.ConnectUser)
```

**Properties**:
- `ID`: Connect user UUID
- `Username`: Connect username
- `Email`: User email
- Other Connect user properties

#### account
Available after `AuthAccount` middleware.
```go
account := c.Locals("account").(*models.Account)
```

**Properties**:
- `ConnectID`: UUID linking to Connect user
- `Balance`: GSALT balance in units
- `Points`: Loyalty points
- `CreatedAt`, `UpdatedAt`: Timestamps

### Endpoint Protection Levels

#### Public Endpoints
No authentication required.
```go
// Example: Health check, voucher listing
router.Get("/health", handler)
router.Get("/vouchers", handler)
```

#### Connect-Only Endpoints
Requires valid Safatanc Connect token.
```go
// Example: Admin functions (if implemented)
router.Post("/admin/action", authMiddleware.AuthConnect, handler)
```

#### Account-Required Endpoints
Requires both Connect authentication AND GSALT account registration.
```go
// Example: Most wallet operations
router.Get("/accounts/me", authMiddleware.AuthConnect, authMiddleware.AuthAccount, handler)
router.Post("/transactions/topup", authMiddleware.AuthConnect, authMiddleware.AuthAccount, handler)
```

### Authentication Headers

All protected endpoints require the Authorization header:

```http
Authorization: Bearer <safatanc-connect-access-token>
```

**Example**:
```http
GET /accounts/me HTTP/1.1
Host: localhost:8080
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Error Handling

The middleware system returns structured error responses:

**401 Unauthorized** - Missing or invalid token:
```json
{
  "success": false,
  "message": "Unauthorized"
}
```

**401 Unauthorized** - Connect user not registered in GSALT:
```json
{
  "success": false,
  "message": "User with connect username john_doe is not registered on GSALT. Please register first."
}
```

### Integration with Safatanc Connect

The authentication system integrates with Safatanc Connect service:

- **Connect Service**: Validates tokens and retrieves user information
- **Account Service**: Links Connect users to GSALT accounts
- **Error Handling**: Provides clear feedback for registration requirements

### Security Features

- **Token Validation**: All tokens validated against Safatanc Connect
- **Account Verification**: Ensures only registered users access wallet features
- **Context Isolation**: User data stored securely in request context
- **Error Clarity**: Clear error messages guide users to correct registration

---

## Voucher Management

### GET /vouchers
Get list of vouchers (public endpoint).

**Query Parameters:**
- `page` (default: 1): Page number to return
- `limit` (default: 10): Number of vouchers per page
- `order` (default: desc): Sort order (asc|desc)
- `order_field` (default: created_at): Field to sort by
- `status` (optional): Filter by status (active|expired|redeemed)

**Response:**
```json
{
  "success": true,
  "data": {
    "page": 1,
    "limit": 10,
    "total_pages": 3,
    "total_items": 25,
    "has_next": true,
    "has_prev": false,
    "items": [
      {
        "id": "uuid",
        "code": "WELCOME2024",
        "name": "Welcome Bonus",
        "description": "Welcome bonus for new users",
        "type": "balance|loyalty_points|discount",
        "value": "50.00",
        "currency": "GSALT",
        "loyalty_points_value": 100,
        "discount_percentage": 10.5,
        "discount_amount": "5.00",
        "max_redeem_count": 1000,
        "current_redeem_count": 245,
        "valid_from": "2024-01-01T00:00:00Z",
        "valid_until": "2024-12-31T23:59:59Z",
        "status": "active",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

### GET /vouchers/:id
Get voucher by ID (public endpoint).

**Parameters:**
- `id` (string): Voucher ID

**Response:** Single voucher object

### GET /vouchers/code/:code
Get voucher by code (public endpoint).

**Parameters:**
- `code` (string): Voucher code

**Response:** Single voucher object

### POST /vouchers/validate/:code
Validate voucher eligibility (public endpoint).

**Parameters:**
- `code` (string): Voucher code

**Response:**
```json
{
  "success": true,
  "data": {
    "valid": true,
    "voucher": {
      "id": "uuid",
      "code": "WELCOME2024",
      "name": "Welcome Bonus",
      "type": "balance",
      "value": "50.00",
      "currency": "GSALT"
    }
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
  "value": "100.00",
  "currency": "GSALT",
  "max_redeem_count": 500,
  "valid_from": "2024-01-01T00:00:00Z",
  "valid_until": "2024-01-31T23:59:59Z"
}
```

**Response:** Created voucher object

### PATCH /vouchers/:id
Update voucher (protected endpoint).

**Headers:** `Authorization: Bearer <token>`

**Parameters:**
- `id` (string): Voucher ID

**Request Body:** Partial voucher update fields

**Response:** Updated voucher object

### DELETE /vouchers/:id
Delete voucher (protected endpoint).

**Headers:** `Authorization: Bearer <token>`

**Parameters:**
- `id` (string): Voucher ID

**Response:**
```json
{
  "success": true,
  "data": null
}
```

---

## Voucher Redemption

### POST /voucher-redemptions/redeem
Redeem voucher (automatically converts to GSALT balance).

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
      "amount_gsalt_units": 5000,
      "currency": "GSALT",
      "status": "completed"
    }
  }
}
```

### POST /voucher-redemptions
Create redemption manually (admin endpoint).

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "voucher_id": "uuid",
  "account_id": "uuid",
  "transaction_id": "uuid"
}
```

**Response:** Created redemption object

### GET /voucher-redemptions/me
Get current user's redemption history.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**
- `page` (default: 1): Page number to return
- `limit` (default: 10): Number of redemptions per page
- `order` (default: desc): Sort order (asc|desc)
- `order_field` (default: redeemed_at): Field to sort by

**Response:**
```json
{
  "success": true,
  "data": {
    "page": 1,
    "limit": 10,
    "total_pages": 2,
    "total_items": 15,
    "has_next": true,
    "has_prev": false,
    "items": [
      {
        "id": "uuid",
        "voucher_id": "uuid",
        "account_id": "uuid",
        "transaction_id": "uuid",
        "redeemed_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

### GET /voucher-redemptions/:id
Get specific redemption by ID.

**Headers:** `Authorization: Bearer <token>`

**Parameters:**
- `id` (string): Redemption ID

**Response:** Single redemption object

### GET /voucher-redemptions/voucher/:voucher_id
Get redemptions by voucher ID (admin endpoint).

**Headers:** `Authorization: Bearer <token>`

**Parameters:**
- `voucher_id` (string): Voucher ID

**Query Parameters:**
- `page` (default: 1): Page number to return
- `limit` (default: 10): Number of redemptions per page
- `order` (default: desc): Sort order (asc|desc)
- `order_field` (default: redeemed_at): Field to sort by

**Response:** Paginated redemption objects

### PATCH /voucher-redemptions/:id
Update redemption.

**Headers:** `Authorization: Bearer <token>`

**Parameters:**
- `id` (string): Redemption ID

**Request Body:**
```json
{
  "transaction_id": "uuid"
}
```

**Response:** Updated redemption object

### DELETE /voucher-redemptions/:id
Delete redemption.

**Headers:** `Authorization: Bearer <token>`

**Parameters:**
- `id` (string): Redemption ID

**Response:**
```json
{
  "success": true,
  "data": null
}
```

---

## Voucher Types & GSALT Conversion

### 1. Balance Voucher (`balance`)
- Adds GSALT balance to user account
- **GSALT Currency**: Directly converted to GSALT units
- **IDR Currency**: Divided by 1000 then converted to GSALT units
- Example: 50 GSALT voucher = 5000 units, 50000 IDR voucher = 5000 units

### 2. Loyalty Points Voucher (`loyalty_points`)
- Adds loyalty points that are converted to GSALT balance
- **Conversion**: 1 point = 1 GSALT unit (0.01 GSALT)
- Example: 100 loyalty points = 100 GSALT units = 1 GSALT

### 3. Discount Voucher (`discount`)
- Provides discount for payments (doesn't change balance)
- **Application**: During payment processing
- Example: 10% discount or 5 GSALT deduction

---

## Exchange Rate System

### Default Exchange Rates
- **1 GSALT = 1000 IDR** (default)
- **USD, EUR, SGD**: Will be added later as needed

### GSALT Units Calculation
```
GSALT Units = GSALT Amount × 100
Example: 10.50 GSALT = 1050 units
```

### Currency Conversion Examples
```
IDR to GSALT: 1,000,000 IDR ÷ 1000 = 1000 GSALT = 100,000 units
USD to GSALT: $100 × exchange_rate = X GSALT = X × 100 units
```

---

## Environment Variables

```env
# Database
DATABASE_URL=postgresql://user:password@localhost:5432/gsalt_core

# Server
PORT=8080

# Safatanc Connect (for authentication)
CONNECT_BASE_URL=https://connect.safatanc.com
CONNECT_CLIENT_ID=your-client-id
CONNECT_CLIENT_SECRET=your-client-secret

# GSALT Configuration
DEFAULT_EXCHANGE_RATE_IDR=1000.00
SUPPORTED_CURRENCIES=IDR,USD,EUR,SGD
GSALT_UNITS_DECIMAL_PLACES=2
```

---

## Response Format

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
  "message": "Error description [ERROR_CODE]"
}
```

### Error Codes
- `INSUFFICIENT_BALANCE`: Insufficient balance
- `DAILY_LIMIT_EXCEEDED`: Daily limit exceeded
- `AMOUNT_BELOW_MINIMUM`: Amount below minimum limit
- `AMOUNT_ABOVE_MAXIMUM`: Amount above maximum limit
- `SELF_TRANSFER_NOT_ALLOWED`: Self transfer not allowed
- `INVALID_STATUS_TRANSITION`: Invalid status transition
- `DUPLICATE_TRANSACTION`: Duplicate transaction

---

## Transaction Status

- `pending`: Transaction is being processed
- `completed`: Transaction completed successfully
- `failed`: Transaction failed
- `cancelled`: Transaction cancelled

## Voucher Status

- `active`: Voucher is active and can be used
- `expired`: Voucher has expired
- `redeemed`: Voucher has reached maximum redemption limit

## Payment Methods

- `GSALT_BALANCE`: Using GSALT balance
- `QRIS`: QRIS payment
- `BANK_TRANSFER`: Bank transfer
- `CREDIT_CARD`: Credit card
- `DEBIT_CARD`: Debit card
- `GOPAY`: GoPay
- `OVO`: OVO
- `DANA`: DANA

---

## Security & Features

- **Atomic Transactions**: Using database transactions for data integrity
- **Row-level Locking**: Prevents race conditions with SELECT FOR UPDATE
- **Idempotency**: Prevents duplicate transactions with external_reference_id
- **Daily Limits**: Daily transaction limits for security
- **Multi-currency Support**: Supports various currencies with exchange rates
- **Validation**: Input validation using go-playground/validator
- **Soft Delete**: Audit trail with soft delete

---

## Support

For questions and support, contact Safatanc Group development team.

**Built with love by Safatanc Group**
