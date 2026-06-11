// Package kfproducer implements the Kafka producer adapter for sending messages to Kafka topics
package kfproducer

// ---------- ProcessTX (ProcessTX)
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
