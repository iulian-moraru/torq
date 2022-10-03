export const DecodeInvoiceErrors = new Map<string, string>([
  ["CHECKSUM_FAILED", "Unable to decode invoice. Checksum failed."],
]);
export const PaymentProcessingErrors = new Map<string, string>([
  ["ALREADY_PAID", "Invoice is already paid"],
  ["INVALID_HASH", "Invalid invoice (invalid hash)"],
  ["INVALID_PAYMENT_REQUEST", "Invalid invoice"],
  ["CHECKSUM_FAILED", "Invalid invoice (checksum failed)"],
  ["UNKNOWN_ERROR", "Unknown error occurred"],
  ["FAILURE_REASON_NO_ROUTE", "Could not find a route to the destination"],
  ["FAILURE_REASON_TIMEOUT", "Payment attempt timed out"],
  ["FAILURE_REASON_ERROR", "Unknown error"],
  ["FAILURE_REASON_INCORRECT_PAYMENT_DETAILS", "Incorrect Payment Details"],
  ["FAILURE_REASON_INCORRECT_PAYMENT_AMOUNT", "Incorrect Payment Amount"],
  ["FAILURE_REASON_PAYMENT_HASH_MISMATCH", "Payment Hash Mismatch"],
  ["FAILURE_REASON_INCORRECT_PAYMENT_REQUEST", "Incorrect Payment Request"],
  ["FAILURE_REASON_INSUFFICIENT_BALANCE", "Insufficient balance"],
  ["FAILURE_REASON_UNKNOWN", "Unknown error"],
  ["AMOUNT_REQUIRED", "Amount must be specified when paying a zero amount invoice"],
  ["AMOUNT_NOT_ALLOWED", "Amount must not be specified when paying a non-zero  amount invoice"],
]);
