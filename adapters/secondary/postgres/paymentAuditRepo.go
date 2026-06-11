package postgres

import (
	"context"
	"fmt"

	outport "github.com/Reddetk/CBTraining/ports/outports"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepo struct {
	db *pgxpool.Pool
}

func NewAuditRepo(db *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{db: db}
}

func (r *AuditRepo) GetTXByID(ctx context.Context, TXID string) (*outport.TXRecord, error) {
	query := `
        SELECT
            t.tx_id,
            t.status,
            t.amount,
            t.currency,
            t.end_to_end_identification,
            t.metadata,

            da.iban,          da.acc_currency,
            dp.bic,           dp.role,   dp.country_of_residence,  dp.name,

            ca.iban,          ca.acc_currency,
            cp.bic,           cp.role,   cp.country_of_residence,  cp.name
        FROM transactions t
        JOIN payment_accounts da ON da.pacc_id = t.debtor_pacc_id
        JOIN parties          dp ON dp.party_id = da.party_id
        JOIN payment_accounts ca ON ca.pacc_id = t.creditor_pacc_id
        JOIN parties          cp ON cp.party_id = ca.party_id
        WHERE t.tx_id = $1
    `

	row := r.db.QueryRow(ctx, query, TXID)

	var tx outport.TXRecord
	tx.DebitorPacc = &outport.PARecord{Party: &outport.PartyRecord{}}
	tx.CreditorPacc = &outport.PARecord{Party: &outport.PartyRecord{}}

	err := row.Scan(
		&tx.TXID,
		&tx.Status,
		&tx.Amount,
		&tx.Currency,
		&tx.EndToEndIdentification,
		&tx.Metadata,

		&tx.DebitorPacc.IBAN,
		&tx.DebitorPacc.AccCurency,
		&tx.DebitorPacc.Party.BIC,
		&tx.DebitorPacc.Party.Role,
		&tx.DebitorPacc.Party.ContryOfResidence,
		&tx.DebitorPacc.Party.Name,

		&tx.CreditorPacc.IBAN,
		&tx.CreditorPacc.AccCurency,
		&tx.CreditorPacc.Party.BIC,
		&tx.CreditorPacc.Party.Role,
		&tx.CreditorPacc.Party.ContryOfResidence,
		&tx.CreditorPacc.Party.Name,
	)
	if err != nil {
		return nil, fmt.Errorf("GetTXByID: %w", err)
	}

	return &tx, nil
}
