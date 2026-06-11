// Package postgres implements the Postgres adapter for storing and retrieving payment transactions and related data from a PostgreSQL database.
package postgres

import (
	"context"
	"fmt"
	"time"

	outport "github.com/Reddetk/CBTraining/ports/outports"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) InsertTX(ctx context.Context, p outport.TXRecord) error {
	now := time.Now().UTC()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("InsertTX begin: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
        INSERT INTO tx_history (
            id, status, amount, currency,
            end_to_end_id, transaction_type,
            debtor_account_id, creditor_account_id,
            created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4,
            $5, $6, $7, $8,
            $9, $9
        )`,
		p.TXID, p.Status, p.Amount, p.Currency,
		p.EndToEndIdentification, p.TransactionType,
		p.DebitorPacc.IBAN, p.CreditorPacc.IBAN,
		p.Metadata,
	)
	if err != nil {
		return fmt.Errorf("InsertTX insert tx_history: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO pending (tx_id, created_at) VALUES ($1, $2)`,
		p.TXID, now,
	)
	if err != nil {
		return fmt.Errorf("InsertTX insert pending: %w", err)
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO outbox (topic, payload)
        VALUES (
            'payment.processing.request',
            jsonb_build_object(
                'txid',            $1,
                'debtorAccount',   $2,
                'creditorAccount', $3
            )
        )`,
		p.TXID, p.DebitorPacc.IBAN, p.CreditorPacc.IBAN,
	)
	if err != nil {
		return fmt.Errorf("InsertTX insert outbox: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("InsertTX commit: %w", err)
	}
	return nil
}

func (r *Repository) LoadPendingPayments(ctx context.Context) ([]outport.TXRecord, error) {
	rows, err := r.db.Query(ctx, `
        SELECT
            t.id,
            t.status,
            t.amount::text,
            t.currency,
            t.end_to_end_id,
            t.transaction_type,

            -- debtor pa_registry
            dpa.id,
            dpa.account_currency,
            -- debtor party_registry
            dp.id,
            dp.role,
            dp.country_of_residence,
            dp.name,

            -- creditor pa_registry
            cpa.id,
            cpa.account_currency,
            -- creditor party_registry
            cp.id,
            cp.role,
            cp.country_of_residence,
            cp.name

        FROM   pending p
        JOIN   tx_history t   ON t.id          = p.tx_id
        JOIN   pa_registry  dpa ON dpa.id      = t.debtor_account_id
        JOIN   party_registry dp ON dp.id      = dpa.party_id
        JOIN   pa_registry  cpa ON cpa.id      = t.creditor_account_id
        JOIN   party_registry cp ON cp.id      = cpa.party_id
        ORDER  BY p.created_at ASC
    `)
	if err != nil {
		return nil, fmt.Errorf("LoadPendingPayments query: %w", err)
	}
	defer rows.Close()

	var result []outport.TXRecord
	for rows.Next() {
		var (
			rec      outport.TXRecord
			debtor   outport.PARecord
			creditor outport.PARecord
			dp       outport.PartyRecord
			cp       outport.PartyRecord
		)

		if err := rows.Scan(
			&rec.TXID,
			&rec.Status,
			&rec.Amount,
			&rec.Currency,
			&rec.EndToEndIdentification,
			&rec.TransactionType,

			&debtor.IBAN,
			&debtor.AccCurency,
			&dp.BIC,
			&dp.Role,
			&dp.ContryOfResidence,
			&dp.Name,

			&creditor.IBAN,
			&creditor.AccCurency,
			&cp.BIC,
			&cp.Role,
			&cp.ContryOfResidence,
			&cp.Name,
		); err != nil {
			return nil, fmt.Errorf("LoadPendingPayments scan: %w", err)
		}

		debtor.Party = &dp
		creditor.Party = &cp
		rec.DebitorPacc = &debtor
		rec.CreditorPacc = &creditor

		result = append(result, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("LoadPendingPayments rows: %w", err)
	}
	return result, nil
}

func (r *Repository) PersistProcessResult(ctx context.Context, txID, result string) error {
	tag, err := r.db.Exec(ctx, `
        UPDATE tx_history
        SET    status     = $2,
               updated_at = NOW()
        WHERE  id = $1
    `, txID, result)
	if err != nil {
		return fmt.Errorf("PersistProcessResult exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("PersistProcessResult: transaction %q not found", txID)
	}
	return nil
}
