package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	outport "github.com/Reddetk/CBTraining/ports/outports"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func (r *Repository) GetTXByID(ctx context.Context, txid string) (*outport.TXRecord, error) {
	query := `
        SELECT
            t.txid,
            t.status,
            t.amount,
            t.currency,
            t.end_to_end_id,
            t.created_at,
            t.updated_at,

            debtor_acc.iban,
            debtor_acc.account_currency,
            debtor_party.bic,
            debtor_party.role,
            debtor_party.country_of_residence,
            debtor_party.name,

            creditor_acc.iban,
            creditor_acc.account_currency,
            creditor_party.bic,
            creditor_party.role,
            creditor_party.country_of_residence,
            creditor_party.name
        FROM tx_history t
        JOIN pa_registry    debtor_acc    ON debtor_acc.iban    = t.debtor_account_id
        JOIN party_registry debtor_party  ON debtor_party.bic   = debtor_acc.party_bic
        JOIN pa_registry    creditor_acc  ON creditor_acc.iban  = t.creditor_account_id
        JOIN party_registry creditor_party ON creditor_party.bic = creditor_acc.party_bic
        WHERE t.txid = $1
    `

	row := r.db.QueryRow(ctx, query, txid)

	var tx outport.TXRecord
	tx.DebtorPacc = &outport.PARecord{Party: &outport.PartyRecord{}}
	tx.CreditorPacc = &outport.PARecord{Party: &outport.PartyRecord{}}

	var updatedAt, createdAt time.Time

	err := row.Scan(
		&tx.TXID,
		&tx.Status,
		&tx.Amount,
		&tx.Currency,
		&tx.EndToEndIdentification,
		&createdAt,
		&updatedAt,

		&tx.DebtorPacc.IBAN,
		&tx.DebtorPacc.AccCurency,
		&tx.DebtorPacc.Party.BIC,
		&tx.DebtorPacc.Party.Role,
		&tx.DebtorPacc.Party.ContryOfResidence,
		&tx.DebtorPacc.Party.Name,

		&tx.CreditorPacc.IBAN,
		&tx.CreditorPacc.AccCurency,
		&tx.CreditorPacc.Party.BIC,
		&tx.CreditorPacc.Party.Role,
		&tx.CreditorPacc.Party.ContryOfResidence,
		&tx.CreditorPacc.Party.Name,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		r.logger.Error("failed to scan transaction", zap.String("txid", txid), zap.Error(err))
		return nil, fmt.Errorf("GetTXByID scan: %w", err)
	}

	metaBytes, err := json.Marshal(map[string]string{
		"created_at": createdAt.UTC().Format(time.RFC3339),
		"updated_at": updatedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return nil, fmt.Errorf("GetTXByID marshal metadata: %w", err)
	}
	tx.Metadata = string(metaBytes)

	return &tx, nil
}
