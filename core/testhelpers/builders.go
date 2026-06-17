//go:build testing
// +build testing

package testhelpers

import (
	"math/rand"
	"strings"
	"time"

	valobj "github.com/Reddetk/CBTraining/core/valObj"
	inport "github.com/Reddetk/CBTraining/ports/inports"
	outport "github.com/Reddetk/CBTraining/ports/outports"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PaymentBuilder позволяет удобно создавать тестовые платежи
// Используется паттерн Builder для улучшения читаемости тестов
type PaymentBuilder struct {
	TXID                  string
	Status                string
	debPartyBIC           string
	debPartyname          string
	debtorIBAN            string
	debitorCur            string
	debtorContry          string
	credPartyBIC          string
	credPartyName         string
	creditorIBAN          string
	creditorCur           string
	creditorContry        string
	amount                string
	currency              string
	txType                string
	endToEndIdentificator string
	metadata              valobj.Metadata
}

func genTXID() string {
	u, _ := uuid.NewRandom()

	return u.String()
}

func genIBAN() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const alnum = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	var b strings.Builder

	b.WriteByte(letters[rand.Intn(len(letters))])
	b.WriteByte(letters[rand.Intn(len(letters))])

	b.WriteByte(byte('0' + rand.Intn(10)))
	b.WriteByte(byte('0' + rand.Intn(10)))

	length := 18 + rand.Intn(9)
	for b.Len() < length {
		b.WriteByte(alnum[rand.Intn(len(alnum))])
	}

	return b.String()
}

func genEndToEndIdentification() string {
	const alnum = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

	// длина от 10 до 20, в пределах до 35
	length := 10 + rand.Intn(11) // 10..20

	out := make([]byte, length)
	for i := range out {
		out[i] = alnum[rand.Intn(len(alnum))]
	}

	return string(out)
}

func genBIC() string {
	const alnum = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	length := 8
	if rand.Intn(2) == 1 {
		length = 11
	}

	out := make([]byte, length)
	for i := range out {
		out[i] = alnum[rand.Intn(len(alnum))]
	}

	return string(out)
}

func genPartyName() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const digits = "0123456789"

	var suffix strings.Builder
	for i := 0; i < 3; i++ {
		suffix.WriteByte(letters[rand.Intn(len(letters))])
	}
	for i := 0; i < 3; i++ {
		suffix.WriteByte(digits[rand.Intn(len(digits))])
	}

	return "Party " + suffix.String()
}

// NewPaymentBuilder создаёт новый builder с дефолтными значениями
func NewPaymentBuilder() *PaymentBuilder {
	return &PaymentBuilder{
		TXID:                  genTXID(),
		debtorIBAN:            genIBAN(),
		creditorIBAN:          genIBAN(),
		Status:                string(valobj.Pending),
		amount:                "300.00",
		currency:              "EUR",
		txType:                "001",
		endToEndIdentificator: genEndToEndIdentification(),
		debPartyBIC:           genBIC(),
		credPartyBIC:          genBIC(),
		debPartyname:          genPartyName(),
		credPartyName:         genPartyName(),
		creditorCur:           "BYN",
		debitorCur:            "BYN",
		debtorContry:          "BY",
		creditorContry:        "BY",
		metadata:              valobj.NewMetadataNow(),
	}
}

func (ps *PaymentBuilder) WithDebtor(IBAN string) *PaymentBuilder {
	ps.debtorIBAN = IBAN
	return ps
}

func (ps *PaymentBuilder) WithCreditor(IBAN string) *PaymentBuilder {
	ps.creditorIBAN = IBAN
	return ps
}

func GenMetadata() valobj.Metadata {
	now := time.Now().UTC()

	// Границы вчерашнего дня
	yesterdayStart := now.AddDate(0, 0, -1).Truncate(24 * time.Hour)
	yesterdayEnd := yesterdayStart.Add(24*time.Hour - time.Millisecond)

	dayDuration := int64(24*time.Hour - time.Millisecond)

	createdAt := yesterdayStart.Add(time.Duration(rand.Int63n(dayDuration))).Truncate(time.Millisecond)

	// updatedAt — случайно после createdAt, но не позже конца вчерашнего дня
	remaining := int64(yesterdayEnd.Sub(createdAt))
	updatedAt := createdAt
	if remaining > 0 {
		updatedAt = createdAt.Add(time.Duration(rand.Int63n(remaining))).Truncate(time.Millisecond)
	}

	return valobj.Metadata{
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func PayreqToTXreq(preq *inport.PaymentRequest, txid string, meta valobj.Metadata) *outport.TXRequest {
	am, _ := decimal.NewFromString(preq.Amount)
	return &outport.TXRequest{
		TXID:                   txid,
		Amount:                 am,
		Status:                 string(valobj.Pending),
		Currency:               preq.Currency,
		EndToEndIdentification: preq.EndToEndIdentification,
		TransactionType:        preq.TransactionType,
		DebtorIBAN:             preq.DebtorPacc.IBAN,
		CreditorIBAN:           preq.CreditorPacc.IBAN,
		Metadata:               meta.String(),
	}
}

func (preq *PaymentBuilder) BuiderToTXrec() outport.TXRecord {
	return outport.TXRecord{
		TXID:                   preq.TXID,
		Amount:                 preq.amount,
		Currency:               preq.currency,
		EndToEndIdentification: preq.endToEndIdentificator,
		Status:                 preq.Status,
		TransactionType:        preq.txType,
		DebtorPacc: &outport.PARecord{
			IBAN:       preq.debtorIBAN,
			AccCurency: preq.debitorCur,
			Party: &outport.PartyRecord{
				BIC:               preq.debPartyBIC,
				Role:              string(valobj.Debtor),
				ContryOfResidence: preq.debtorContry,
				Name:              preq.debPartyname,
			},
		},
		CreditorPacc: &outport.PARecord{
			IBAN:       preq.creditorIBAN,
			AccCurency: preq.creditorCur,
			Party: &outport.PartyRecord{
				BIC:               preq.credPartyBIC,
				Role:              string(valobj.Creditor),
				ContryOfResidence: preq.creditorContry,
				Name:              preq.credPartyName,
			},
		},
		Metadata: preq.metadata.String(),
	}
}

func (pb *PaymentBuilder) CrIBAN() string {
	return pb.creditorIBAN
}

func (pb *PaymentBuilder) DbIBAN() string {
	return pb.debtorIBAN
}

// Build создаёт и возвращает конфигурированный PaymentRequest
func (pb *PaymentBuilder) Build() *inport.PaymentRequest {
	return &inport.PaymentRequest{
		Amount:                 pb.amount,
		Currency:               pb.currency,
		EndToEndIdentification: pb.endToEndIdentificator,
		TransactionType:        pb.txType,
		Debtor: &inport.PaymentPartyDTO{
			Name:              pb.debPartyname,
			BIC:               pb.debPartyBIC,
			Role:              "debtor",
			ContryOfResidence: pb.debtorContry,
		},
		DebtorPacc: &inport.PaymentAccountDTO{
			IBAN:       pb.debtorIBAN,
			AccCurency: pb.debitorCur,
		},
		Creditor: &inport.PaymentPartyDTO{
			Name:              pb.debPartyname,
			BIC:               pb.debPartyBIC,
			Role:              "creditor",
			ContryOfResidence: pb.creditorContry,
		},
		CreditorPacc: &inport.PaymentAccountDTO{
			IBAN:       pb.creditorIBAN,
			AccCurency: pb.debitorCur,
		},
	}
}

// BuildMany создаёт N платежей с указанными парами IBAN
func BuildMany(pairs []IBANPair) ([]*inport.PaymentRequest, []*PaymentBuilder) {
	result := make([]*inport.PaymentRequest, len(pairs))
	pb := make([]*PaymentBuilder, len(pairs))
	for i, pair := range pairs {
		pb[i] = NewPaymentBuilder().WithCreditor(pair.Creditor).WithDebtor(pair.Debtor)
		result[i] = pb[i].Build()
	}
	return result, pb
}

// IBANPair описывает пару дебитор-кредитор для тестирования
type IBANPair struct {
	Debtor   string // IBAN дебитора
	Creditor string // IBAN кредитора
}

// ============================================================================
// СПЕЦИАЛИЗИРОВАННЫЕ BUILDERS
// ============================================================================

// BuildIndependentPair создаёт пару независимых платежей (без пересечений)
func BuildIndependentPair() ([]*inport.PaymentRequest, []*PaymentBuilder) {
	return BuildMany([]IBANPair{
		{Debtor: genIBAN(), Creditor: genIBAN()},
		{Debtor: genIBAN(), Creditor: genIBAN()},
	})
}

// BuildDependentChain создаёт цепочку платежей с общими IBAN
func BuildDependentChain(length int) ([]*inport.PaymentRequest, []*PaymentBuilder) {
	if length < 2 {
		panic("chain length must be at least 2")
	}

	ibans := make([]string, length+1)
	for i := range ibans {
		ibans[i] = genIBAN()
	}

	// Создаём цепочку: IBAN0->IBAN1, IBAN1->IBAN2, ..., IBANn-1->IBANn
	pairs := make([]IBANPair, length)
	for i := 0; i < length; i++ {
		pairs[i] = IBANPair{
			Debtor:   ibans[i],
			Creditor: ibans[i+1],
		}
	}

	return BuildMany(pairs)
}

// BuildConcurrentPressure создаёт N платежей все пытающихся доступить одну ЦС
func BuildConcurrentPressure(count int) ([]*inport.PaymentRequest, []*PaymentBuilder) {
	// Все платежи конвертируют деньги из разных счётов в один IBAN "SHARED"
	pairs := make([]IBANPair, count)
	sharedAcc := genIBAN()
	for i := 0; i < count; i++ {
		pairs[i] = IBANPair{
			Debtor:   genIBAN(),
			Creditor: sharedAcc,
		}
	}
	return BuildMany(pairs)
}

// ============================================================================
// HELPER ФУНКЦИИ
// ============================================================================

// MergePaymentRequests объединяет несколько slice'ов платежей в один
func MergePaymentRequests(slices ...[]*inport.PaymentRequest) []*inport.PaymentRequest {
	totalLen := 0
	for _, slice := range slices {
		totalLen += len(slice)
	}

	result := make([]*inport.PaymentRequest, 0, totalLen)
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}
