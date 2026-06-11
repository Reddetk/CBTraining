// Package postgres implements the Postgres adapter for storing and retrieving payment transactions and related data from a PostgreSQL database.
package postgres

// | Request | Called from | Moment |
// | -------------------- | -------------------------------- | ----------------------- |
// | InsertTX | HTTP handler | Receiving payment |
// | ProcessTX | runWorker in Dispatcher | Before sending to Kafka |
// | RestorePending | main / Dispatcher.Restore() | Starting the service |
// | PersistProcessResult | Kafka consumer (independent pool) | Receiving PaymentResult |

// ------------ InsertTX
// BEGIN;
// INSERT INTO tx_history (
//     id, status, amount, currency,
//     end_to_end_id, transaction_type,
//     debtor_account_id, creditor_account_id,
//     created_at, updated_at
// ) VALUES (
//     $1, $2, $3, $4,
//     $5, $6, $7, $8,
//     $9, $9
// );
// INSERT INTO pending (tx_id, created_at) VALUES ($1, $9);
// COMMIT;

// ---------- ProcessTX
// BEGIN;
// UPDATE tx_history
// SET    status     = $2,  -- 'processing'
//        updated_at = $3
// WHERE  id = $1;
//
// DELETE FROM pending
// WHERE  tx_id = $1;
//
// INSERT INTO outbox (topic, payload)
// VALUES (
//     'payment.processing.request',
//     jsonb_build_object('txid', $1, 'debtorAccount', $4, 'creditorAccount', $5)
// );
// COMMIT;

// ---- RestorePending
// SELECT p.tx_id,
//        t.debtor_account_id,
//        t.creditor_account_id,
//        t.amount, t.currency,
//        t.transaction_type, t.end_to_end_id
// FROM   pending p
// JOIN   tx_history t ON t.id = p.tx_id
// ORDER  BY p.created_at ASC;

// ---- PersistProcessResult
// UPDATE tx_history
// SET    status     = $2,  -- 'completed' | 'failed' from  PaymentResult.Result
//        updated_at = $3
// WHERE  id = $1;          -- PaymentResult.TXID
