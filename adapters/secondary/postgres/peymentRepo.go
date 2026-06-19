// Package postgres implements the Postgres adapter for storing and retrieving payment transactions and related data from a PostgreSQL database.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Reddetk/CBTraining/logger"
	outport "github.com/Reddetk/CBTraining/ports/outports"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Repository struct {
	db     *pgxpool.Pool
	logger logger.Logger
}

func New(db *pgxpool.Pool, logger logger.Logger) *Repository {
	return &Repository{db: db, logger: logger}
}

func (r *Repository) InsertTX(ctx context.Context, p outport.TXRecord) error {
	now := time.Now().UTC()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		r.logger.Error("failed to begin transaction", zap.Error(err))
		return fmt.Errorf("InsertTX begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Upsert debtor party
	_, err = tx.Exec(ctx, `
        INSERT INTO party_registry (bic, role, country_of_residence, name)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (bic) DO NOTHING`,
		p.DebtorPacc.Party.BIC,
		p.DebtorPacc.Party.Role,
		p.DebtorPacc.Party.ContryOfResidence,
		p.DebtorPacc.Party.Name,
	)
	if err != nil {
		r.logger.Error("failed to upsert debtor party", zap.String("bic", p.DebtorPacc.Party.BIC), zap.Error(err))
		return fmt.Errorf("InsertTX upsert debtor party: %w", err)
	}

	// 2. Upsert creditor party
	_, err = tx.Exec(ctx, `
        INSERT INTO party_registry (bic, role, country_of_residence, name)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (bic) DO NOTHING`,
		p.CreditorPacc.Party.BIC,
		p.CreditorPacc.Party.Role,
		p.CreditorPacc.Party.ContryOfResidence,
		p.CreditorPacc.Party.Name,
	)
	if err != nil {
		r.logger.Error("failed to upsert creditor party", zap.String("bic", p.CreditorPacc.Party.BIC), zap.Error(err))
		return fmt.Errorf("InsertTX upsert creditor party: %w", err)
	}

	// 3. Upsert debtor account
	_, err = tx.Exec(ctx, `
        INSERT INTO pa_registry (iban, account_currency, party_bic)
        VALUES ($1, $2, $3)
        ON CONFLICT (iban) DO NOTHING`,
		p.DebtorPacc.IBAN, p.DebtorPacc.AccCurency, p.DebtorPacc.Party.BIC,
	)
	if err != nil {
		r.logger.Error("failed to upsert debtor account", zap.String("iban", p.DebtorPacc.IBAN), zap.Error(err))
		return fmt.Errorf("InsertTX upsert debtor account: %w", err)
	}

	// 4. Upsert creditor account
	_, err = tx.Exec(ctx, `
        INSERT INTO pa_registry (iban, account_currency, party_bic)
        VALUES ($1, $2, $3)
        ON CONFLICT (iban) DO NOTHING`,
		p.CreditorPacc.IBAN, p.CreditorPacc.AccCurency, p.CreditorPacc.Party.BIC,
	)
	if err != nil {
		r.logger.Error("failed to upsert creditor account", zap.String("iban", p.CreditorPacc.IBAN), zap.Error(err))
		return fmt.Errorf("InsertTX upsert creditor account: %w", err)
	}

	// 5. Вставка транзакции
	_, err = tx.Exec(ctx, `
        INSERT INTO tx_history (
            txid, status, amount, currency,
            end_to_end_id, transaction_type,
            debtor_account_id, creditor_account_id,
            created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4,
            $5, $6, $7, $8,
            $9, $10
        )`,
		p.TXID, p.Status, p.Amount, p.Currency,
		p.EndToEndIdentification, p.TransactionType,
		p.DebtorPacc.IBAN, p.CreditorPacc.IBAN,
		now, now,
	)
	if err != nil {
		r.logger.Error("failed to insert into tx_history", zap.String("txid", p.TXID), zap.Error(err))
		return fmt.Errorf("InsertTX insert tx_history: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO pending (txid, created_at) VALUES ($1, $2)`,
		p.TXID, now,
	)
	if err != nil {
		r.logger.Error("failed to insert into pending", zap.String("txid", p.TXID), zap.Error(err))
		return fmt.Errorf("InsertTX insert pending: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		r.logger.Error("failed to commit transaction", zap.String("txid", p.TXID), zap.Error(err))
		return fmt.Errorf("InsertTX commit: %w", err)
	}

	r.logger.Info("inserted new transaction", zap.String("txid", p.TXID))
	return nil
}

func (r *Repository) LoadPendingPayments(ctx context.Context) ([]outport.TXRecord, error) {
	rows, err := r.db.Query(ctx, `
	SELECT
		t.txid,
		t.status,
		t.amount::text,
		t.currency,
		t.end_to_end_id,
		t.transaction_type,

		json_build_object(
        'created_at', to_json(t.created_at),
        'updated_at', to_json(t.updated_at)
    	)::text AS metadata,

		-- debtor pa_registry
		dpa.iban,
		dpa.account_currency,
		-- debtor party_registry
		dp.bic,
		dp.role,
		dp.country_of_residence,
		dp.name,

		-- creditor pa_registry
		cpa.iban,
		cpa.account_currency,
		-- creditor party_registry
		cp.bic,
		cp.role,
		cp.country_of_residence,
		cp.name

	FROM   pending p
	JOIN   tx_history    t   ON t.txid     = p.txid
	JOIN   pa_registry   dpa ON dpa.iban   = t.debtor_account_id
	JOIN   party_registry dp ON dp.bic     = dpa.party_bic
	JOIN   pa_registry   cpa ON cpa.iban   = t.creditor_account_id
	JOIN   party_registry cp ON cp.bic     = cpa.party_bic
	ORDER  BY p.created_at ASC
    `)
	if err != nil {
		r.logger.Error("failed to query pending payments", zap.Error(err))
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
			&rec.Metadata,

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
			r.logger.Error("failed to scan pending payment", zap.Error(err))
			return nil, fmt.Errorf("LoadPendingPayments scan: %w", err)
		}

		debtor.Party = &dp
		creditor.Party = &cp
		rec.DebtorPacc = &debtor
		rec.CreditorPacc = &creditor

		result = append(result, rec)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating pending payments", zap.Error(err))
		return nil, fmt.Errorf("LoadPendingPayments rows: %w", err)
	}

	r.logger.Info("loaded pending payments", zap.Int("count", len(result)))
	return result, nil
}

func (r *Repository) PersistProcessResult(ctx context.Context, txID, result string) error { // TODO Atomicity
	tag, err := r.db.Exec(ctx, `
        UPDATE tx_history
        SET    status     = $2,
               updated_at = NOW()
        WHERE  txid = $1
    `, txID, result)
	if err != nil {
		r.logger.Error("failed to update transaction status", zap.String("txid", txID), zap.String("status", result), zap.Error(err))
		return fmt.Errorf("PersistProcessResult exec: %w", err)
	}
	if tag.RowsAffected() == 0 {
		r.logger.Error("transaction not found", zap.String("txid", txID))
		return fmt.Errorf("PersistProcessResult: transaction %q not found", txID)
	}

	r.logger.Info("updated transaction status", zap.String("txid", txID), zap.String("status", result))
	return nil
}

func (r *Repository) ProcessTX(ctx context.Context, p outport.TXRequest) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		r.logger.Error("failed to begin transaction", zap.String("txid", p.TXID), zap.Error(err))
		return fmt.Errorf("ProcessTX begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	now := time.Now().UTC()

	// UPDATE tx_history
	_, err = tx.Exec(ctx, `
        UPDATE tx_history
        SET    status     = $2,
               updated_at = $3
        WHERE  txid       = $1
    `,
		p.TXID,
		"processing",
		now,
	)
	if err != nil {
		r.logger.Error("failed to update transaction status to processing", zap.String("txid", p.TXID), zap.Error(err))
		return fmt.Errorf("ProcessTX update tx_history: %w", err)
	}

	// DELETE FROM pending
	_, err = tx.Exec(ctx, `
        DELETE FROM pending
        WHERE  txid = $1
    `,
		p.TXID,
	)
	if err != nil {
		r.logger.Error("failed to delete from pending", zap.String("txid", p.TXID), zap.Error(err))
		return fmt.Errorf("ProcessTX delete pending: %w", err)
	}

	// INSERT INTO outbox
	payload, err := json.Marshal(map[string]any{
		"txid":            p.TXID,
		"status":          p.Status,
		"amount":          p.Amount,
		"currency":        p.Currency,
		"endToEndId":      p.EndToEndIdentification,
		"transactionType": p.TransactionType,
		"debtorIBAN":      p.DebtorIBAN,
		"creditorIBAN":    p.CreditorIBAN,
		"metadata":        p.Metadata,
	})
	if err != nil {
		r.logger.Error("marshal outbox payload: ", zap.Error(err))
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	_, err = tx.Exec(ctx, `
    INSERT INTO outbox (txid, topic, payload, created_at)
    VALUES ($1, 'payment.processing.request', $2, $3)
`,
		p.TXID, payload, now,
	)
	if err != nil {
		r.logger.Error("failed to insert into outbox", zap.String("txid", p.TXID), zap.Error(err))
		return fmt.Errorf("ProcessTX insert outbox: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		r.logger.Error("failed to commit transaction", zap.String("txid", p.TXID), zap.Error(err))
		return fmt.Errorf("ProcessTX commit: %w", err)
	}

	r.logger.Info("processed transaction", zap.String("txid", p.TXID))
	return nil
}
