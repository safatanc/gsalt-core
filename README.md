# GSalt Core - Payment Gateway & Digital Wallet System

GSalt Core is Safatanc Group's payment gateway and digital wallet system that supports topup, transfer, payment, and voucher redemption with GSALT (Safatanc Global Loyalty Token) system. This system functions as a multi-currency payment gateway and digital wallet with automatic conversion to GSALT units.

## Table of Contents

- [Overview](#overview)
- [System Architecture](#system-architecture)
- [Payment Flow Diagram](#payment-flow-diagram)
- [Features](#features)
- [Tech Stack](#tech-stack)
- [GSALT System](#gsalt-system)
- [Payment Methods](#payment-methods)
- [Xendit Integration](#xendit-integration)
- [Payment Instructions](#payment-instructions)
- [Installation](#installation)
- [API Documentation](#api-documentation)
  - [Authentication](#authentication)
  - [Health Check](#health-check)
  - [Account Management](#account-management)
  - [Transaction Management](#transaction-management)
  - [Webhook Endpoints](#webhook-endpoints)
- [Authentication & Middleware](#authentication--middleware)
- [Voucher Management](#voucher-management)
- [Voucher Redemption](#voucher-redemption)
- [Environment Variables](#environment-variables)
- [Database Schema](#database-schema)
- [Error Handling](#error-handling)
- [Security Features](#security--features)
- [Support](#support)

## Overview

GSalt Core provides a comprehensive digital wallet and payment gateway solution with the following core capabilities:

- **Multi-currency Digital Wallet** with GSALT token system
- **External Payment Gateway** integration via Xendit
- **Real-time Transaction Processing** with atomic operations
- **Voucher & Loyalty System** with automatic conversions
- **Webhook-based Payment Confirmation** for automated processing

## System Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend/     │    │   GSalt Core    │    │   External      │
│   Mobile App    │    │   Backend       │    │   Services      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │  1. Topup Request     │                       │
         ├──────────────────────►│                       │
         │                       │  2. Payment Request   │
         │                       ├──────────────────────►│ Xendit API
         │                       │                       │
         │  3. Payment Instructions                      │
         │◄──────────────────────┤  3. Payment Response  │
         │                       │◄──────────────────────┤
         │                       │                       │
         │  4. User Payment      │                       │
         │  (QR/VA/E-wallet)     │                       │
         ├───────────────────────┼──────────────────────►│
         │                       │                       │
         │                       │  5. Webhook Event     │
         │                       │◄──────────────────────┤
         │                       │                       │
         │  6. Balance Updated   │                       │
         │◄──────────────────────┤                       │
```

## Payment Flow Diagram

### External Payment Flow (QRIS/VA/E-wallet)

```mermaid
graph TD
    A[User Request Topup] --> B[Create Pending Transaction]
    B --> C[Call Xendit API]
    C --> D[Store Payment Instructions]
    D --> E[Return Payment Instructions to User]
    E --> F[User Makes Payment]
    F --> G[Xendit Webhook]
    G --> H{Payment Status}
    H -->|Success| I[Confirm Transaction]
    H -->|Failed| J[Reject Transaction]
    I --> K[Update Account Balance]
    K --> L[Transaction Completed]
    J --> M[Transaction Failed]
```

### Internal Payment Flow (GSALT Balance)

```mermaid
graph TD
    A[User Request Payment] --> B[Check Balance]
    B --> C{Sufficient Balance?}
    C -->|Yes| D[Deduct Balance]
    C -->|No| E[Return Error]
    D --> F[Create Transaction]
    F --> G[Transaction Completed]
```

### Transfer Flow

```mermaid
graph TD
    A[User Request Transfer] --> B[Validate Accounts]
    B --> C[Check Source Balance]
    C --> D{Sufficient Balance?}
    D -->|Yes| E[Create Transfer Out]
    D -->|No| F[Return Error]
    E --> G[Create Transfer In]
    G --> H[Update Balances]
    H --> I[Link Transactions]
    I --> J[Transfer Completed]
```

## Features

- **Digital Wallet**: Balance management in GSALT units with multi-currency conversion
- **Payment Gateway**: Payment processing through various methods (QRIS, Bank Transfer, GSALT Balance)
- **Multi-Currency Support**: Supports IDR, USD, EUR, SGD with automatic exchange rates
- **Transaction Processing**: Topup, transfer, payment with atomic operations
- **Voucher System**: Voucher management with various types and GSALT conversion
- **Xendit Integration**: External payment processing via Xendit Payment Gateway
- **GSALT Exchange**: 1 GSALT = 100 units (2 decimal places), default 1000 IDR = 1 GSALT
- **Payment Instructions**: Automatic storage of payment details for frontend consumption
- **Webhook Processing**: Real-time payment confirmation via Xendit webhooks
- **Security**: Row-level locking, idempotency, daily limits, and validation

## Tech Stack

- **Backend**: Go (Fiber framework)
- **Database**: PostgreSQL with GORM
- **Authentication**: JWT via Safatanc Connect
- **Payment Gateway**: Xendit Payment Gateway
- **Dependency Injection**: Google Wire
- **Validation**: go-playground/validator
- **Migration**: Database migrations support

## GSALT System

### GSALT Units
- **1 GSALT = 100 units** (for 2 decimal precision)
- **Default Exchange Rate**: 1000 IDR = 1 GSALT
- **Balance Storage**: Stored in GSALT units (int64)
- **Supported Currencies**: IDR, USD, EUR, SGD

### Conversion Examples
```
GSALT to Units: 10.50 GSALT = 1050 units
IDR to GSALT: 1,000,000 IDR ÷ 1000 = 1000 GSALT = 100,000 units
USD to GSALT: $100 × exchange_rate = X GSALT = X × 100 units
```

## Payment Methods

### Internal Payment Methods
- `GSALT_BALANCE`: Pay using GSALT balance

### External Payment Methods (Xendit)
- `QRIS`: QRIS payment
- `VIRTUAL_ACCOUNT`: Bank transfer (Virtual Account)
- `EWALLET`: E-wallet payments
- `RETAIL_OUTLET`: Retail outlet payments
- `CARDLESS_CREDIT`: Cardless credit payments
- `CARD`: Card payments

### Legacy Payment Methods
- `CREDIT_CARD`: Credit card
- `DEBIT_CARD`: Debit card
- `GOPAY`: GoPay
- `OVO`: OVO
- `DANA`: DANA

## Xendit Integration

GSalt Core integrates with Xendit Payment Gateway to process external payments for topup transactions. The integration supports multiple payment methods and handles the complete payment lifecycle.

### Supported Xendit Payment Methods

#### 1. QRIS
- Indonesian QR Code payment standard
- Instant payment confirmation
- Works with all QRIS-compatible apps

#### 2. Virtual Account
- Bank transfer via virtual account numbers
- Supported banks: BCA, BNI, BRI, Mandiri, BSI, Permata
- Automatic payment detection

#### 3. E-Wallet
- Digital wallet payments
- Supported providers: OVO, DANA, GoPay, LinkAja, ShopeePay
- Real-time payment confirmation

#### 4. Retail Outlet
- Over-the-counter payments at retail stores
- Supported outlets based on Xendit configuration

#### 5. Cardless Credit
- Cardless credit payment options
- Various credit providers supported

#### 6. Card Payments
- Credit and debit card processing
- Secure card tokenization

### Transaction Status Management

- **PENDING**: External payment transactions start in pending status
- **COMPLETED**: Payment confirmed via webhook, balance updated
- **FAILED**: Payment failed or expired
- **CANCELLED**: Payment cancelled or expired

## Payment Instructions

GSalt Core automatically stores payment instructions from Xendit in the transaction record as JSON data. This eliminates the need for frontend applications to make additional API calls to retrieve payment details.

### What's Included in Payment Instructions

**For QRIS Payments:**
- QR Code string for rendering payment QR
- Payment amount and currency
- Expiry time for countdown timer
- Payment request ID for tracking

**For Virtual Account Payments:**
- Bank code (BCA, BNI, BRI, etc.)
- Virtual account number for transfer
- Payment amount and instructions
- Expiry time

**For E-Wallet Payments:**
- Provider name (OVO, DANA, GoPay, etc.)
- Checkout URL for redirection
- QR code (if available)
- Deep links for mobile apps

### Benefits

1. **Single API Call**: All payment information available in topup response
2. **Offline Capability**: Payment instructions stored locally, no need for additional Xendit calls
3. **Persistent Storage**: Instructions remain available even after page refresh
4. **Frontend Ready**: JSON format can be directly consumed by frontend applications
5. **Multi-Method Support**: Unified structure for all payment methods

### Usage Example

```javascript
// Frontend JavaScript example
const topupResponse = await fetch('/transactions/topup', {
  method: 'POST',
  body: JSON.stringify({
    amount_gsalt: "10.00",
    payment_method: "QRIS"
  })
});

const data = await topupResponse.json();

if (data.success && data.data.payment_instructions) {
  const instructions = data.data.payment_instructions;
  
  if (instructions.qr_code) {
    // Render QR code for QRIS payment
    renderQRCode(instructions.qr_code);
  } else if (instructions.virtual_account) {
    // Display VA number for bank transfer
    displayVANumber(instructions.virtual_account.account_number);
  } else if (instructions.ewallet?.checkout_url) {
    // Redirect to e-wallet checkout
    window.location.href = instructions.ewallet.checkout_url;
  }
  
  // Show expiry countdown
  if (instructions.expiry_time) {
    startCountdown(instructions.expiry_time);
  }
}
```

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

4. Configure Xendit credentials in `.env`:
```env
# Xendit Configuration
XENDIT_SECRET_KEY=your-xendit-secret-key
XENDIT_WEBHOOK_TOKEN=your-webhook-verification-token
XENDIT_ENVIRONMENT=sandbox # or "live" for production
XENDIT_BASE_URL=https://api.xendit.co # default Xendit API URL
XENDIT_CALLBACK_URL=https://your-app.com/api/v1/webhooks/xendit/payment # webhook callback URL
XENDIT_SUCCESS_URL=https://your-app.com/payment/success # payment success redirect URL
XENDIT_FAILURE_URL=https://your-app.com/payment/failure # payment failure redirect URL
APP_BASE_URL=http://localhost:8080 # your app URL for webhooks
```

5. Run database migrations
```bash
# Run your migration files in migrations/ folder
```

6. Generate Wire dependencies
```bash
cd injector && wire
```

7. Run application
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

### POST /accounts
Create a new GSALT account for authenticated Safatanc Connect user.

**Headers:** `Authorization: Bearer <token>`

**Request Body:** No request body required. Account is created using information from the authenticated Connect user.

**Response:**
```json
{
  "success": true,
  "data": {
    "connect_id": "uuid",
    "balance": 0,
    "points": 0,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

**Notes:**
- Requires valid Safatanc Connect authentication
- New accounts start with 0 balance (in GSALT units) and 0 loyalty points
- Returns error if account already exists for the Connect user
- This endpoint only requires `AuthConnect` middleware (not `AuthAccount`)

**Error Responses:**
- `400 Bad Request`: If account already exists or Connect user not found
- `401 Unauthorized`: If Connect token is missing or invalid

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
Process balance topup via external payment or direct credit.

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

**Request Body (Auto-calculated payment amount):**
```json
{
  "amount_gsalt": "10.00",
  "payment_currency": "IDR",
  "payment_method": "QRIS"
}
```

**Notes:**
- `amount_gsalt`: Amount in GSALT (will be converted to units)
- `payment_amount`: Actual payment amount (optional, auto-calculated if not provided)
- `payment_currency`: Payment currency (optional, defaults to IDR)
- `payment_method`: Payment method (optional, defaults to QRIS)
- **External Payment Methods** (`QRIS`, `VIRTUAL_ACCOUNT`, `EWALLET`, `RETAIL_OUTLET`, `CARDLESS_CREDIT`, `CARD`): Creates pending transaction, requires Xendit payment
- **Direct Credit**: Other payment methods immediately complete transaction

**Response (External Payment):**
```json
{
  "success": true,
  "data": {
    "transaction": {
      "id": "uuid",
      "type": "topup",
      "amount_gsalt_units": 1000,
      "payment_amount": 10000,
      "payment_currency": "IDR",
      "payment_method": "QRIS",
      "status": "pending",
      "external_reference_id": "xendit-payment-request-id"
    },
    "payment_instructions": {
      "payment_request_id": "xendit-payment-request-id",
      "status": "PENDING",
      "amount": 10000,
      "currency": "IDR",
      "payment_method": "QR_CODE",
      "qr_code": "00020101021226820014com.xendit...",
      "expiry_time": "2024-01-01T10:00:00Z",
      "created_at": "2024-01-01T00:00:00Z"
    }
  }
}
```

**Payment Instructions by Method:**

*QRIS:*
```json
"payment_instructions": {
  "payment_request_id": "xendit-id-123",
  "qr_code": "00020101021226820014com.xendit...",
  "payment_method": "QR_CODE",
  "amount": 10000,
  "currency": "IDR",
  "expiry_time": "2024-01-01T10:00:00Z"
}
```

*Virtual Account:*
```json
"payment_instructions": {
  "payment_request_id": "xendit-id-123",
  "virtual_account": {
    "bank_code": "BCA",
    "account_number": "1234567890"
  },
  "payment_method": "VIRTUAL_ACCOUNT",
  "amount": 10000,
  "currency": "IDR",
  "expiry_time": "2024-01-01T10:00:00Z"
}
```

*E-Wallet:*
```json
"payment_instructions": {
  "payment_request_id": "xendit-id-123",
  "ewallet": {
    "provider": "OVO",
    "checkout_url": "https://checkout.ovo.id/payment/..."
  },
  "payment_method": "EWALLET",
  "amount": 10000,
  "currency": "IDR",
  "expiry_time": "2024-01-01T10:00:00Z"
}
```

**Response (Direct Credit):**
```json
{
  "success": true,
  "data": {
    "transaction": {
      "id": "uuid",
      "type": "topup",
      "amount_gsalt_units": 1000,
      "payment_amount": 10000,
      "payment_currency": "IDR",
      "payment_method": "DIRECT_CREDIT",
      "status": "completed"
    }
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

### POST /transactions/:id/confirm
Confirm pending payment transaction (admin or webhook use).

**Headers:** `Authorization: Bearer <token>`

**Parameters:**
- `id` (string): Transaction ID

**Request Body:**
```json
{
  "external_payment_id": "xendit-payment-id"
}
```

**Response:** Updated transaction object with status "completed"

### POST /transactions/:id/reject
Reject pending payment transaction.

**Headers:** `Authorization: Bearer <token>`

**Parameters:**
- `id` (string): Transaction ID

**Request Body:**
```json
{
  "reason": "Payment failed or expired"
}
```

**Response:** Updated transaction object with status "failed"

---

## Webhook Endpoints

### POST /webhooks/xendit/payment
Handle payment notifications from Xendit (no authentication required).

**Request Body:**
```json
{
  "id": "webhook-event-id",
  "event": "payment.completed",
  "payment_request_id": "xendit-payment-request-id",
  "transaction_id": "gsalt-transaction-uuid",
  "status": "completed",
  "amount": 10000,
  "currency": "IDR",
  "payment_method": "QRIS",
  "paid_at": "2024-01-01T00:00:00Z",
  "created_at": "2024-01-01T00:00:00Z"
}
```

**Supported Events:**
- `payment.completed`: Payment successful, transaction confirmed
- `payment.failed`: Payment failed, transaction rejected
- `payment.expired`: Payment expired, transaction rejected

**Response:**
```json
{
  "status": "success",
  "message": "Webhook processed"
}
```

**Notes:**
- This endpoint automatically processes the webhook and updates transaction status
- Returns success even if processing fails to prevent Xendit retries
- Webhook signature verification should be implemented for production

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
// Example: Health check, voucher listing, webhooks
router.Get("/health", handler)
router.Get("/vouchers", handler)
router.Post("/webhooks/xendit/payment", handler)
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

# Xendit Configuration
XENDIT_SECRET_KEY=your-xendit-secret-key
XENDIT_WEBHOOK_TOKEN=your-webhook-verification-token
XENDIT_ENVIRONMENT=sandbox
XENDIT_BASE_URL=https://api.xendit.co
XENDIT_CALLBACK_URL=https://your-app.com/api/v1/webhooks/xendit/payment
XENDIT_SUCCESS_URL=https://your-app.com/payment/success
XENDIT_FAILURE_URL=https://your-app.com/payment/failure
APP_BASE_URL=http://localhost:8080

# GSALT Configuration
DEFAULT_EXCHANGE_RATE_IDR=1000.00
SUPPORTED_CURRENCIES=IDR,USD,EUR,SGD
GSALT_UNITS_DECIMAL_PLACES=2
```

---

## Database Schema

### Core Tables

#### accounts
```sql
CREATE TABLE accounts (
    connect_id UUID PRIMARY KEY,
    balance BIGINT NOT NULL DEFAULT 0,
    points INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP NULL
);
```

#### transactions
```sql
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(connect_id),
    type VARCHAR(50) NOT NULL,
    amount_gsalt_units BIGINT NOT NULL,
    currency VARCHAR(5) DEFAULT 'GSALT',
    exchange_rate_idr DECIMAL(10,2) DEFAULT 1000.00,
    payment_amount BIGINT NULL,
    payment_currency VARCHAR(3) NULL,
    payment_method VARCHAR(50) NULL,
    payment_instructions JSONB NULL,
    status VARCHAR(20) DEFAULT 'pending',
    description TEXT NULL,
    source_account_id UUID NULL,
    destination_account_id UUID NULL,
    voucher_code VARCHAR(50) NULL,
    external_reference_id VARCHAR(255) NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP NULL,
    deleted_at TIMESTAMP NULL
);
```

#### vouchers
```sql
CREATE TABLE vouchers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT NULL,
    type VARCHAR(20) NOT NULL,
    value DECIMAL(10,2) NULL,
    currency VARCHAR(5) NULL,
    loyalty_points_value INTEGER NULL,
    discount_percentage DECIMAL(5,2) NULL,
    discount_amount DECIMAL(10,2) NULL,
    max_redeem_count INTEGER DEFAULT 1,
    current_redeem_count INTEGER DEFAULT 0,
    valid_from TIMESTAMP NOT NULL,
    valid_until TIMESTAMP NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP NULL
);
```

#### voucher_redemptions
```sql
CREATE TABLE voucher_redemptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    voucher_id UUID NOT NULL REFERENCES vouchers(id),
    account_id UUID NOT NULL REFERENCES accounts(connect_id),
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    redeemed_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP NULL
);
```

### Indexes

```sql
-- Transaction indexes
CREATE INDEX idx_transactions_account_id ON transactions(account_id);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);
CREATE INDEX idx_transactions_external_ref ON transactions(external_reference_id);

-- Voucher indexes
CREATE INDEX idx_vouchers_code ON vouchers(code);
CREATE INDEX idx_vouchers_status ON vouchers(status);
CREATE INDEX idx_vouchers_valid_dates ON vouchers(valid_from, valid_until);

-- Redemption indexes
CREATE INDEX idx_redemptions_voucher_id ON voucher_redemptions(voucher_id);
CREATE INDEX idx_redemptions_account_id ON voucher_redemptions(account_id);
```

---

## Error Handling

### Response Format

#### Success Response
```json
{
  "success": true,
  "data": { /* response data */ }
}
```

#### Error Response
```json
{
  "success": false,
  "message": "Error description [ERROR_CODE]"
}
```

### Error Codes

#### Business Logic Errors
- `INSUFFICIENT_BALANCE`: Insufficient balance for transaction
- `DAILY_LIMIT_EXCEEDED`: Daily transaction limit exceeded
- `AMOUNT_BELOW_MINIMUM`: Transaction amount below minimum limit
- `AMOUNT_ABOVE_MAXIMUM`: Transaction amount above maximum limit
- `SELF_TRANSFER_NOT_ALLOWED`: Cannot transfer to same account
- `INVALID_STATUS_TRANSITION`: Invalid transaction status transition
- `DUPLICATE_TRANSACTION`: Duplicate transaction (idempotency check)

#### Validation Errors
- `INVALID_ACCOUNT_ID`: Invalid account ID format
- `INVALID_AMOUNT`: Invalid amount format or value
- `INVALID_CURRENCY`: Unsupported currency
- `INVALID_PAYMENT_METHOD`: Unsupported payment method

#### Authentication Errors
- `UNAUTHORIZED`: Missing or invalid authentication token
- `ACCOUNT_NOT_REGISTERED`: Connect user not registered in GSALT

#### External Service Errors
- `XENDIT_API_ERROR`: Xendit API request failed
- `XENDIT_WEBHOOK_INVALID`: Invalid webhook signature or payload
- `PAYMENT_REQUEST_FAILED`: Failed to create payment request

#### System Errors
- `INTERNAL_SERVER_ERROR`: Unexpected server error
- `DATABASE_ERROR`: Database operation failed
- `NETWORK_ERROR`: Network connectivity issue

### HTTP Status Codes

- `200 OK`: Successful request
- `400 Bad Request`: Invalid request data or business logic error
- `401 Unauthorized`: Authentication required or failed
- `403 Forbidden`: Access denied
- `404 Not Found`: Resource not found
- `409 Conflict`: Duplicate transaction or conflict
- `422 Unprocessable Entity`: Validation error
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error
- `502 Bad Gateway`: External service error
- `503 Service Unavailable`: Service temporarily unavailable

---

## Security & Features

### Security Features

- **Atomic Transactions**: Using database transactions for data integrity
- **Row-level Locking**: Prevents race conditions with SELECT FOR UPDATE
- **Idempotency**: Prevents duplicate transactions with external_reference_id
- **Daily Limits**: Daily transaction limits for security (configurable)
- **Input Validation**: Comprehensive input validation using go-playground/validator
- **Authentication**: JWT-based authentication via Safatanc Connect
- **Webhook Security**: Configurable webhook signature verification
- **Soft Delete**: Audit trail with soft delete for data integrity

### Transaction Features

- **Multi-currency Support**: Supports various currencies with automatic exchange rates
- **Payment Gateway Integration**: Secure external payment processing via Xendit
- **Transaction Status Management**: Comprehensive status tracking and transitions
- **Balance Management**: Real-time balance updates with locking mechanisms
- **Transfer System**: Peer-to-peer transfers with automatic transaction linking

### Voucher Features

- **Flexible Voucher Types**: Balance, loyalty points, and discount vouchers
- **Automatic Conversion**: Seamless conversion to GSALT units
- **Usage Tracking**: Real-time redemption count and status management
- **Expiry Management**: Time-based voucher validity

### Performance Features

- **Database Indexing**: Optimized queries with proper indexing
- **Pagination**: Efficient pagination for large datasets
- **Caching**: Configurable caching for frequently accessed data
- **Connection Pooling**: Database connection pooling for optimal performance

---

## Transaction Status

- `pending`: Transaction is being processed (external payments)
- `completed`: Transaction completed successfully
- `failed`: Transaction failed
- `cancelled`: Transaction cancelled

## Voucher Status

- `active`: Voucher is active and can be used
- `expired`: Voucher has expired
- `redeemed`: Voucher has reached maximum redemption limit

---

## Xendit Integration Details

### XenditService Features

1. **Payment Request Creation**
   - Creates payment requests for various methods
   - Automatic conversion from GSALT to IDR
   - Configurable expiry times
   - Customer information mapping

2. **Webhook Processing**
   - Automatic transaction confirmation/rejection
   - Event-driven payment status updates
   - Webhook signature validation (configurable)

3. **Supported Payment Methods**
   - QRIS with QR code generation
   - Virtual Account with bank selection
   - E-wallet with provider selection

4. **Error Handling**
   - Comprehensive error responses
   - Fallback mechanisms
   - Transaction rollback on failures

### Configuration

The XenditService requires configuration through environment variables or dependency injection:

```go
type XenditConfig struct {
    SecretKey    string // Xendit API secret key
    WebhookToken string // Webhook verification token
    CallbackURL  string // Webhook callback URL
    SuccessURL   string // Payment success redirect URL
    FailureURL   string // Payment failure redirect URL
    Environment  string // "sandbox" or "live"
    BaseURL      string // Xendit API base URL
}
```

### Security Considerations

1. **Webhook Verification**: Implement webhook signature validation in production
2. **Secret Management**: Store Xendit credentials securely
3. **HTTPS**: Use HTTPS for webhook endpoints in production
4. **Rate Limiting**: Implement rate limiting for webhook endpoints

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

## Support

For questions and support, contact Safatanc Group development team.

**Built with love by Safatanc Group**
