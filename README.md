# GSALT Core - Payment Gateway & Digital Wallet System

GSalt Core is Safatanc Group's payment gateway and digital wallet system that supports topup, transfer, payment, withdrawal, and voucher redemption with GSALT (Safatanc Global Loyalty Token) system. This system functions as a multi-currency payment gateway and digital wallet with automatic conversion to GSALT units.

## Table of Contents

- [Overview](#overview)
- [System Architecture](#system-architecture)
- [Payment Flow Diagram](#payment-flow-diagram)
- [Features](#features)
- [Tech Stack](#tech-stack)
- [GSALT System](#gsalt-system)
- [Payment Methods](#payment-methods)
- [Flip Integration](#flip-integration)
- [Payment Instructions](#payment-instructions)
- [Withdrawal System](#withdrawal-system)
- [Installation](#installation)
- [API Documentation](#api-documentation)
  - [Authentication](#authentication)
  - [Health Check](#health-check)
  - [Account Management](#account-management)
  - [Transaction Management](#transaction-management)
  - [Withdrawal Management](#withdrawal-management)
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
- **External Payment Gateway** integration via Flip for Business (BigFlip)
- **Real-time Transaction Processing** with atomic operations
- **Voucher & Loyalty System** with automatic conversions
- **Withdrawal System** with bank disbursement via Flip
- **Webhook-based Payment Confirmation** for automated processing
- **Comprehensive Payment Methods** supporting all major Indonesian payment channels

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
         │                       ├──────────────────────►│ Flip API V3
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
         │                       │                       │
         │  7. Withdrawal Request│                       │
         ├──────────────────────►│                       │
         │                       │  8. Disbursement      │
         │                       ├──────────────────────►│ Flip Disbursement
         │                       │                       │
         │  9. Withdrawal Status │                       │
         │◄──────────────────────┤                       │
```

## Payment Flow Diagram

### External Payment Flow (QRIS/VA/E-wallet/Credit Card/Retail)

```mermaid
graph TD
    A[User Request Topup] --> B[Calculate Fee]
    B --> C[Create Pending Transaction]
    C --> D[Call Flip API V3]
    D --> E[Store Payment Instructions]
    E --> F[Return Payment Instructions to User]
    F --> G[User Makes Payment]
    G --> H[Flip Webhook]
    H --> I{Payment Status}
    I -->|Success| J[Confirm Transaction]
    I -->|Failed| K[Reject Transaction]
    J --> L[Update Account Balance]
    L --> M[Transaction Completed]
    K --> N[Transaction Failed]
```

### Withdrawal Flow

```mermaid
graph TD
    A[User Request Withdrawal] --> B[Validate Bank Account]
    B --> C[Check Balance & Limits]
    C --> D{Sufficient Balance?}
    D -->|Yes| E[Deduct Balance]
    D -->|No| F[Return Error]
    E --> G[Create Disbursement]
    G --> H{Disbursement Success?}
    H -->|Yes| I[Transaction Processing]
    H -->|No| J[Refund Balance]
    I --> K[Bank Transfer]
    K --> L[Transaction Completed]
    J --> M[Transaction Failed]
```

## Features

- **Digital Wallet**: Balance management in GSALT units with multi-currency conversion
- **Payment Gateway**: Payment processing through various methods with user-paid fees
- **Multi-Currency Support**: Supports IDR, USD, EUR, SGD with automatic exchange rates
- **Transaction Processing**: Topup, transfer, payment, withdrawal with atomic operations
- **Voucher System**: Voucher management with various types and GSALT conversion
- **Flip Integration**: External payment processing via Flip for Business V3 API
- **Withdrawal System**: Bank disbursement via Flip with real-time status tracking
- **GSALT Exchange**: 1 GSALT = 100 units (2 decimal places), default 1000 IDR = 1 GSALT
- **Payment Instructions**: Automatic storage of payment details for frontend consumption
- **Webhook Processing**: Real-time payment confirmation via Flip webhooks
- **Fee Management**: Transparent fee structure with user-paid processing fees
- **Unified Payment API**: Single robust payment endpoint with comprehensive error handling
- **Security**: Row-level locking, idempotency, daily limits, and validation

## Tech Stack

- **Backend**: Go (Fiber framework)
- **Database**: PostgreSQL with GORM
- **Authentication**: JWT via Safatanc Connect
- **Payment Gateway**: Flip for Business (BigFlip) V3 API
- **Disbursement**: Flip Disbursement API
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

### External Payment Methods (Flip V3)

#### Virtual Account
- `VA_BCA`: BCA Virtual Account
- `VA_BNI`: BNI Virtual Account  
- `VA_BRI`: BRI Virtual Account
- `VA_MANDIRI`: Mandiri Virtual Account
- `VA_CIMB`: CIMB Virtual Account
- `VA_PERMATA`: Permata Virtual Account
- `VA_BSI`: BSI Virtual Account
- `VA_DANAMON`: Danamon Virtual Account
- `VA_MAYBANK`: Maybank Virtual Account

#### QRIS
- `QRIS`: Indonesian QR Code payment standard

#### E-Wallet
- `EWALLET_OVO`: OVO
- `EWALLET_DANA`: DANA
- `EWALLET_GOPAY`: GoPay
- `EWALLET_LINKAJA`: LinkAja
- `EWALLET_SHOPEEPAY`: ShopeePay

#### Credit Card
- `CREDIT_CARD`: Credit card payments
  - Visa, Mastercard, JCB, American Express

#### Debit Card
- `DEBIT_CARD`: Debit card payments
  - Same providers as credit card

#### Retail Outlets
- `RETAIL_ALFAMART`: Alfamart
- `RETAIL_INDOMARET`: Indomaret
- `RETAIL_CIRCLEK`: Circle K
- `RETAIL_LAWSON`: Lawson
- `RETAIL_DANDANPAY`: DandanPay

#### Direct Debit
- `DIRECT_DEBIT`: Direct debit from bank account
  - BCA, BNI, BRI, Mandiri

#### Bank Transfer
- `BANK_TRANSFER`: Manual bank transfer

## Flip Integration

GSalt Core integrates with Flip for Business (BigFlip) V3 API to process external payments for topup transactions and handle withdrawals via disbursement. The integration supports comprehensive payment methods and handles the complete payment lifecycle with transparent fee structure.

### Service Architecture

The Flip integration is organized into two main service layers:

#### FlipService (API Layer)
- **General API Methods**: Balance, banks, maintenance info
- **Accept Payment V2**: Bill creation, payment processing, status tracking
- **Disbursement API**: Bank account validation, money transfer, withdrawal processing
- **Business Logic Methods**: Enhanced payment processing with detailed payment method information

#### TransactionService (Business Layer)
- Uses FlipService for external payment processing
- Handles topup transactions with automatic fee calculation
- Manages payment confirmations via webhooks
- Processes withdrawals with GSALT to IDR conversion

### Enhanced Features

#### Payment Processing
- **CreateTopupPaymentWithDetails**: Creates payment bills with comprehensive payment method information
- **ConvertPaymentResponseToJSON**: Converts payment responses to JSON for storage
- **Payment Method Details**: Detailed breakdown for each payment method (QR codes, VA numbers, checkout URLs)
- **Automatic Fee Calculation**: Transparent fee structure with user-paid processing fees
- **Expiry Management**: Configurable payment expiry times (1-168 hours)

#### Withdrawal System
- **ValidateBankAccount**: Real-time bank account validation
- **CheckDisbursementAvailability**: Service maintenance status checking
- **CreateGSALTDisbursement**: GSALT-specific disbursement with automatic conversion
- **GetSupportedBanks**: Dynamic list of operational banks

#### Data Models
- **Enhanced Models**: Complete payment method support with detailed field definitions
- **Error Handling**: Structured error responses with detailed messages
- **Type Safety**: Comprehensive enum definitions for payment methods and statuses

### Fee Structure (User-Paid)

All fees are paid by the user (sender/buyer) and calculated automatically:

#### Virtual Account
- **Fee**: 4 GSALT (4,000 IDR)
- **Banks**: BCA, BNI, BRI, Mandiri, CIMB, Permata, BSI, Danamon, Maybank

#### QRIS
- **Fee**: 0.7% of amount (minimum 2 GSALT)
- **Example**: 100 GSALT topup = 0.7 GSALT fee (minimum 2 GSALT)

#### E-Wallet
- **Fee**: 5 GSALT (5,000 IDR)
- **Providers**: OVO, DANA, GoPay, LinkAja, ShopeePay

#### Credit/Debit Card
- **Fee**: 2.9% of amount (minimum 10 GSALT)
- **Example**: 100 GSALT topup = 2.9 GSALT fee (minimum 10 GSALT)

#### Retail Outlets
- **Fee**: 5 GSALT (5,000 IDR)
- **Outlets**: Alfamart, Indomaret, Circle K, Lawson, DandanPay

#### Direct Debit
- **Fee**: 3 GSALT (3,000 IDR)
- **Banks**: BCA, BNI, BRI, Mandiri

### V3 API Features

- **JSON Request/Response**: Modern JSON-based API communication
- **Item Details**: Detailed breakdown of purchase items and fees
- **Reference ID**: Transaction tracking and idempotency
- **Charge Fee**: Transparent fee charging to users
- **Enhanced Payment Methods**: Complete payment method configuration
- **Checkout Flow**: Seamless checkout experience
- **Payment Method Details**: Comprehensive payment information for all supported methods
- **Automatic Conversion**: GSALT to IDR conversion with configurable exchange rates
- **Service Availability**: Real-time maintenance and service status checking

### Code Organization

#### API Methods (FlipService)
```go
// General API
GetBalance(ctx) - Retrieve account balance
GetBanks(ctx) - Get supported banks list
GetMaintenanceInfo(ctx) - Check service status

// Payment Processing
CreateBill(ctx, req) - Create payment bill
GetBill(ctx, billID) - Get bill status
CreateTopupPaymentWithDetails(ctx, req) - Enhanced payment creation

// Disbursement
BankAccountInquiry(ctx, req) - Validate bank account
CreateDisbursement(ctx, req) - Process withdrawal
CreateGSALTDisbursement(ctx, req) - GSALT-specific withdrawal
```

#### Helper Methods
```go
// Payment Method Configuration
setPaymentMethodFields() - Configure payment method specific fields
createPaymentMethodDetails() - Generate detailed payment information

// Data Conversion
ConvertPaymentResponseToJSON() - Convert responses to JSON
ConvertGSALTToIDR() - Currency conversion

// Validation
ValidateBankAccount() - Bank account validation
CheckDisbursementAvailability() - Service availability check
```

### Transaction Status Management

- **PENDING**: External payment transactions start in pending status
- **PROCESSING**: Withdrawal transactions being processed by bank
- **COMPLETED**: Payment confirmed via webhook, balance updated
- **FAILED**: Payment failed or expired
- **CANCELLED**: Payment cancelled or expired

## Withdrawal System

GSalt Core provides a comprehensive withdrawal system that allows users to convert their GSALT balance to IDR and transfer to Indonesian bank accounts via Flip's disbursement service.

### Withdrawal Features

- **Bank Account Validation**: Real-time validation of recipient bank accounts
- **Balance Checking**: Available balance calculation excluding pending withdrawals
- **Daily Limits**: Configurable daily withdrawal limits for security
- **Real-time Status**: Track withdrawal status from pending to completed
- **Supported Banks**: All major Indonesian banks supported by Flip
- **Automatic Conversion**: GSALT to IDR conversion (1 GSALT = 1,000 IDR)
- **Maintenance Check**: Automatic service availability verification

### Supported Banks for Withdrawal

All major Indonesian banks are supported including:
- BCA, BNI, BRI, Mandiri
- CIMB Niaga, Permata, BSI
- Danamon, Maybank, and others

### Withdrawal Process

1. **Validation**: Bank account validation via Flip API
2. **Balance Check**: Verify sufficient balance and daily limits
3. **Service Check**: Confirm disbursement service availability
4. **Deduction**: Deduct amount from user balance
5. **Disbursement**: Create disbursement request to Flip
6. **Tracking**: Monitor status until completion
7. **Completion**: Bank transfer completed (1-3 business days)

### Withdrawal Limits

- **Minimum**: 1 GSALT (1,000 IDR)
- **Maximum**: 25,000 GSALT (25,000,000 IDR) per transaction
- **Daily Limit**: 100,000 GSALT (100,000,000 IDR)

## Payment Instructions

GSalt Core automatically stores payment instructions from Flip in the transaction record as JSON data. This eliminates the need for frontend applications to make additional API calls to retrieve payment details.

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

**For Credit/Debit Card Payments:**
- Secure payment form URL
- Card input fields configuration
- 3D Secure handling
- Payment completion flow

**For Retail Outlet Payments:**
- Outlet name and locations
- Payment code for cashier
- Instructions for payment
- Expiry time

### Benefits

1. **Single API Call**: All payment information available in topup response
2. **Offline Capability**: Payment instructions stored locally, no need for additional Flip calls
3. **Persistent Storage**: Instructions remain available even after page refresh
4. **Frontend Ready**: JSON format can be directly consumed by frontend applications
5. **Multi-Method Support**: Unified structure for all payment methods
6. **Fee Transparency**: Clear breakdown of amount and fees

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
  
  // Display fee information
  if (instructions.fee_amount) {
    displayFeeInfo(instructions.fee_amount, instructions.total_amount);
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

4. Configure Flip credentials in `.env`:
```env
# Flip Configuration
FLIP_SECRET_KEY=your-flip-secret-key
FLIP_WEBHOOK_TOKEN=your-webhook-verification-token
FLIP_ENVIRONMENT=sandbox # or "production" for production
FLIP_PRODUCTION_URL=https://bigflip.id/api/v3 # production API URL
FLIP_SANDBOX_URL=https://bigflip.id/big_sandbox_api/v3 # sandbox API URL
FLIP_CALLBACK_URL=https://your-app.com/api/v1/webhooks/flip/payment # webhook callback URL
FLIP_SUCCESS_URL=https://your-app.com/payment/success # payment success redirect URL
FLIP_FAILURE_URL=https://your-app.com/payment/failure # payment failure redirect URL
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

### GET /transactions
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

### PUT /transactions/:id
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
- **External Payment Methods** (`QRIS`, `VA_*`, `EWALLET_*`, `RETAIL_*`, `CREDIT_CARD`, `DEBIT_CARD`): Creates pending transaction, requires Flip payment
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
      "external_reference_id": "flip-payment-request-id"
    },
    "payment_instructions": {
      "payment_request_id": "flip-payment-request-id",
      "status": "PENDING",
      "amount": 10000,
      "currency": "IDR",
      "payment_method": "QR_CODE",
      "qr_code": "00020101021226820014com.flip...",
      "expiry_time": "2024-01-01T10:00:00Z",
      "created_at": "2024-01-01T00:00:00Z"
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
Process payment (can use GSALT balance or external payment with comprehensive payment method support).

**Headers:** `Authorization: Bearer <token>`

**Note:** This endpoint has been optimized with robust error handling and unified payment processing for all supported payment methods.

**Request Body:**
```json
{
  "amount_gsalt": "25.00",
  "payment_method": "GSALT_BALANCE",
  "description": "Payment for service",
  "external_reference_id": "merchant-ref-123"
}
```

**Request Body (External Payment with Fee):**
```json
{
  "amount_gsalt": "100.00",
  "payment_method": "QRIS",
  "description": "External payment",
  "external_reference_id": "merchant-ref-456"
}
```

**Supported Payment Methods:**
- `GSALT_BALANCE`: Direct payment using GSALT balance (no fees)
- `QRIS`: QR Code payment (0.7% fee, min 2 GSALT)
- `VA_BCA`, `VA_BNI`, `VA_BRI`, `VA_MANDIRI`, etc.: Virtual Account (4 GSALT fee)
- `EWALLET_OVO`, `EWALLET_DANA`, `EWALLET_GOPAY`, etc.: E-wallet (5 GSALT fee)
- `CREDIT_CARD`: Credit card payment (2.9% fee, min 10 GSALT)
- `DEBIT_CARD`: Debit card payment (2.9% fee, min 10 GSALT)
- `RETAIL_ALFAMART`, `RETAIL_INDOMARET`, etc.: Retail outlets (5 GSALT fee)
- `DIRECT_DEBIT`: Direct debit (3 GSALT fee)

**Response (GSALT Balance):**
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

**Response (External Payment):**
```json
{
  "success": true,
  "data": {
    "transaction": {
      "id": "uuid",
      "type": "payment",
      "amount_gsalt_units": 10000,
      "total_amount_gsalt_units": 10070,
      "fee_amount_gsalt_units": 70,
      "payment_method": "QRIS",
      "status": "pending"
    },
    "payment_instructions": {
      "payment_request_id": "flip-payment-id",
      "qr_code": "00020101021226820014com.flip...",
      "amount": 100700,
      "currency": "IDR",
      "fee_amount": 700,
      "total_amount": 100700,
      "expiry_time": "2024-01-01T10:00:00Z"
    }
  }
}
```

### GET /transactions/payment/methods
Get supported payment methods with calculated fees.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**
- `amount` (optional): Amount for fee calculation (default: 100)

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "method": "VA_BCA",
      "name": "Virtual Account BCA",
      "type": "virtual_account",
      "fee": 4000,
      "description": "Transfer melalui Virtual Account BCA"
    },
    {
      "method": "QRIS",
      "name": "QRIS",
      "type": "qris",
      "fee": 2000,
      "description": "Bayar dengan scan QR Code"
    },
    {
      "method": "EWALLET_OVO",
      "name": "OVO",
      "type": "ewallet",
      "fee": 5000,
      "description": "Bayar dengan OVO"
    }
  ]
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
  "external_payment_id": "flip-payment-id"
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

## Withdrawal Management

### POST /transactions/withdrawal
Process withdrawal to Indonesian bank account.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "amount_gsalt": "100.00",
  "bank_code": "BCA",
  "account_number": "1234567890",
  "recipient_name": "John Doe",
  "description": "Withdrawal to bank account",
  "external_reference_id": "withdrawal-ref-123"
}
```

**Request Body Fields:**
- `amount_gsalt`: Amount in GSALT to withdraw (will be converted to IDR)
- `bank_code`: Bank code (BCA, BNI, BRI, MANDIRI, etc.)
- `account_number`: Recipient bank account number
- `recipient_name`: Account holder name (must match bank records)
- `description`: Optional description for the withdrawal
- `external_reference_id`: Optional unique reference ID

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "type": "withdrawal",
    "amount_gsalt_units": 10000,
    "payment_amount": 100000,
    "payment_currency": "IDR",
    "status": "processing",
    "description": "Withdrawal to bank account",
    "external_reference_id": "flip-disbursement-id",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

**Notes:**
- Minimum withdrawal: 1 GSALT (1,000 IDR)
- Maximum withdrawal: 25,000 GSALT (25,000,000 IDR) per transaction
- Daily limit: 100,000 GSALT (100,000,000 IDR)
- Bank transfer usually takes 1-3 business days
- Account validation is performed before processing

### GET /transactions/withdrawal/banks
Get list of supported banks for withdrawal.

**Headers:** `Authorization: Bearer <token>`

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "code": "BCA",
      "name": "Bank Central Asia",
      "available": true
    },
    {
      "code": "BNI",
      "name": "Bank Negara Indonesia",
      "available": true
    },
    {
      "code": "BRI",
      "name": "Bank Rakyat Indonesia",
      "available": true
    },
    {
      "code": "MANDIRI",
      "name": "Bank Mandiri",
      "available": true
    }
  ]
}
```

### GET /transactions/withdrawal/balance
Get available balance for withdrawal.

**Headers:** `Authorization: Bearer <token>`

**Response:**
```json
{
  "success": true,
  "data": {
    "balance_gsalt_units": 50000,
    "balance_gsalt": "500.00",
    "balance_idr": "500000.00"
  }
}
```

**Notes:**
- `balance_gsalt_units`: Available balance in GSALT units
- `balance_gsalt`: Available balance in GSALT decimal format
- `balance_idr`: Equivalent IDR amount (1 GSALT = 1,000 IDR)

### POST /transactions/withdrawal/:id/status
Check withdrawal transaction status.

**Headers:** `Authorization: Bearer <token>`

**Parameters:**
- `id` (string): Transaction ID

**Response:**
```json
{
  "success": true,
  "data": {
    "transaction_id": "uuid",
    "status": "processing",
    "amount_gsalt": "100.00",
    "amount_idr": "100000.00",
    "bank_code": "BCA",
    "account_number": "1234567890",
    "recipient_name": "John Doe",
    "disbursement_id": "flip-disbursement-id",
    "created_at": "2024-01-01T00:00:00Z",
    "estimated_completion": "2024-01-03T00:00:00Z"
  }
}
```

**Status Values:**
- `pending`: Withdrawal request created, awaiting processing
- `processing`: Disbursement sent to bank, awaiting completion
- `completed`: Bank transfer completed successfully
- `failed`: Withdrawal failed, balance refunded

### POST /transactions/withdrawal/validate-bank-account
Validate bank account before withdrawal.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "bank_code": "BCA",
  "account_number": "1234567890"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "valid": true,
    "account_holder_name": "John Doe",
    "bank_name": "Bank Central Asia"
  }
}
```

**Error Response:**
```json
{
  "success": false,
  "message": "Invalid bank account or account not found"
}
```

---

## Webhook Endpoints

### POST /transactions/webhook/flip
Handle payment notifications from Flip (no authentication required).

**Request Body:**
```json
{
  "id": "webhook-event-id",
  "event": "payment.completed",
  "payment_request_id": "flip-payment-request-id",
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
- Returns success even if processing fails to prevent Flip retries
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
router.Post("/webhooks/flip/payment", handler)
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

Configure the following environment variables in your `.env` file:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=gsalt_core
DB_SSL_MODE=disable

# Server Configuration
APP_PORT=8080
APP_ENV=development
APP_BASE_URL=http://localhost:8080

# JWT Configuration
JWT_SECRET=your-jwt-secret-key
JWT_EXPIRES_IN=24h

# Safatanc Connect Configuration
CONNECT_API_URL=https://connect.safatanc.com/api
CONNECT_CLIENT_ID=your-connect-client-id
CONNECT_CLIENT_SECRET=your-connect-client-secret

# Flip Configuration
FLIP_SECRET_KEY=your-flip-secret-key
FLIP_WEBHOOK_TOKEN=your-webhook-verification-token
FLIP_ENVIRONMENT=sandbox # or "production"
FLIP_PRODUCTION_URL=https://bigflip.id/api/v3
FLIP_SANDBOX_URL=https://bigflip.id/big_sandbox_api/v3
FLIP_CALLBACK_URL=https://your-app.com/api/v1/webhooks/flip/payment
FLIP_SUCCESS_URL=https://your-app.com/payment/success
FLIP_FAILURE_URL=https://your-app.com/payment/failure

# Redis Configuration (Optional)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json
```

---

## Database Schema

The system uses PostgreSQL with the following main tables:

### accounts
- `connect_id` (UUID, Primary Key): Link to Safatanc Connect user
- `balance` (BIGINT): Balance in GSALT units
- `points` (BIGINT): Loyalty points
- `created_at`, `updated_at` (TIMESTAMP)

### transactions
- `id` (UUID, Primary Key): Transaction ID
- `account_id` (UUID): Reference to account
- `type` (VARCHAR): Transaction type
- `amount_gsalt_units` (BIGINT): Amount in GSALT units
- `status` (VARCHAR): Transaction status
- `payment_method` (VARCHAR): Payment method used
- `external_reference_id` (VARCHAR): External payment reference
- `payment_instructions` (JSONB): Payment instructions from Flip
- `created_at`, `updated_at` (TIMESTAMP)

### vouchers
- `id` (UUID, Primary Key): Voucher ID
- `code` (VARCHAR, Unique): Voucher code
- `name` (VARCHAR): Voucher name
- `type` (VARCHAR): Voucher type
- `value` (DECIMAL): Voucher value
- `currency` (VARCHAR): Currency
- `max_redeem_count` (INTEGER): Maximum redemptions
- `current_redeem_count` (INTEGER): Current redemptions
- `valid_from`, `valid_until` (TIMESTAMP): Validity period
- `status` (VARCHAR): Voucher status
- `created_at`, `updated_at` (TIMESTAMP)

### voucher_redemptions
- `id` (UUID, Primary Key): Redemption ID
- `voucher_id` (UUID): Reference to voucher
- `account_id` (UUID): Reference to account
- `transaction_id` (UUID): Reference to transaction
- `redeemed_at` (TIMESTAMP): Redemption timestamp

---

## Error Handling

The system implements comprehensive error handling:

### HTTP Status Codes
- `200 OK`: Successful request
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request data
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Access denied
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource conflict (e.g., duplicate)
- `422 Unprocessable Entity`: Validation errors
- `500 Internal Server Error`: Server error

### Error Response Format
```json
{
  "success": false,
  "message": "Error description",
  "errors": {
    "field_name": ["Validation error message"]
  }
}
```

### Validation Errors
The system uses `go-playground/validator` for request validation with detailed error messages.

---

## Security Features

### Authentication & Authorization
- **JWT-based Authentication**: Secure token-based authentication
- **Role-based Access Control**: Different access levels for different user types
- **Token Expiration**: Configurable token expiration times
- **Refresh Token Support**: Secure token refresh mechanism

### Data Protection
- **SQL Injection Prevention**: Parameterized queries with GORM
- **XSS Protection**: Input sanitization and output encoding
- **CORS Configuration**: Configurable cross-origin resource sharing
- **Rate Limiting**: API rate limiting to prevent abuse

### Transaction Security
- **Atomic Operations**: Database transactions for data consistency
- **Idempotency Keys**: Prevent duplicate transactions
- **Row-level Locking**: Prevent race conditions
- **Audit Trail**: Complete transaction history and logging

### Payment Security
- **Webhook Verification**: Signature verification for Flip webhooks
- **Secure API Keys**: Encrypted storage of sensitive credentials
- **PCI Compliance**: Secure handling of payment data
- **Fraud Detection**: Basic fraud detection mechanisms

---

## Support

For questions and support, contact Safatanc Group development team:

- **Email**: dev@safatanc.com
- **Documentation**: [GSalt Core Documentation](https://docs.gsalt.safatanc.com)
- **API Reference**: [GSalt Core API](https://api.gsalt.safatanc.com/docs)

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

### License

This project is proprietary software owned by Safatanc Group.

**Built with love by Safatanc Group**