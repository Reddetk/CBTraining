package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
	metadata              Metadata
}

func genTXID() string {
	u, _ := uuid.NewRandom()

	return u.String()[:34]
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

// Metadata represents creation/update timestamps -------------
type Metadata struct {
	CreatedAt time.Time `json:"created_at"` // Unix RFC3339 timestamp in milliseconds
	UpdatedAt time.Time `json:"updated_at"` // Unix RFC3339 timestamp in milliseconds
}

// NewMetadata creates Metadata with explicit timestamps
func NewMetadata(input string) (Metadata, error) {
	var mt Metadata
	err := json.Unmarshal([]byte(input), &mt)
	if err != nil {
		return Metadata{}, fmt.Errorf("metadata not valid")
	}
	if mt.UpdatedAt.Before(mt.CreatedAt) {
		return Metadata{}, fmt.Errorf("Err Metadata Updated Before Created")
	}

	return mt, nil
}

// NewMetadataNow creates Metadata with current time for both timestamps
func NewMetadataNow() Metadata {
	return Metadata{CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
}

// Touch returns new Metadata with updated updatedAt -- immutable update
func (m Metadata) Touch() Metadata {
	return Metadata{
		CreatedAt: m.CreatedAt,
		UpdatedAt: time.Now().UTC(),
	}
}

func (m Metadata) String() string {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

// NewPaymentBuilder создаёт новый builder с дефолтными значениями
func NewPaymentBuilder() *PaymentBuilder {
	return &PaymentBuilder{
		TXID:                  genTXID(),
		debtorIBAN:            genIBAN(),
		creditorIBAN:          genIBAN(),
		Status:                "pending",
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
		metadata:              NewMetadataNow(),
	}
}

func (pb *PaymentBuilder) WithDebtor(IBAN string) *PaymentBuilder {
	pb.debtorIBAN = IBAN
	return pb
}

func (pb *PaymentBuilder) WithCreditor(IBAN string) *PaymentBuilder {
	pb.creditorIBAN = IBAN
	return pb
}

func GenMetadata() Metadata {
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

	return Metadata{
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func (pb *PaymentBuilder) CrIBAN() string {
	return pb.creditorIBAN
}

func (pb *PaymentBuilder) DbIBAN() string {
	return pb.debtorIBAN
}

type PaymentRequest struct {
	Amount                 string             `json:"amount" binding:"required"`
	Currency               string             `json:"currency" binding:"required"`
	EndToEndIdentification string             `json:"endToEndIdentification" binding:"required"`
	TransactionType        string             `json:"transactionType" binding:"required"`
	Debtor                 *PaymentPartyDTO   `json:"debtor" binding:"required"`
	DebtorPacc             *PaymentAccountDTO `json:"debtorPacc" binding:"required"`
	Creditor               *PaymentPartyDTO   `json:"creditor" binding:"required"`
	CreditorPacc           *PaymentAccountDTO `json:"creditorPacc" binding:"required"`
}

type PaymentPartyDTO struct {
	BIC               string `json:"bic" binding:"required"`
	Role              string `json:"role" binding:"required"`
	ContryOfResidence string `json:"contryOfResidence" binding:"required"`
	Name              string `json:"name" binding:"required"`
}

type PaymentAccountDTO struct {
	IBAN       string `json:"iban" binding:"required"`
	AccCurency string `json:"accCurency" binding:"required"`
}

// Build создаёт и возвращает конфигурированный PaymentRequest
func (pb *PaymentBuilder) Build() *PaymentRequest {
	return &PaymentRequest{
		Amount:                 pb.amount,
		Currency:               pb.currency,
		EndToEndIdentification: pb.endToEndIdentificator,
		TransactionType:        pb.txType,
		Debtor: &PaymentPartyDTO{
			Name:              pb.debPartyname,
			BIC:               pb.debPartyBIC,
			Role:              "debtor",
			ContryOfResidence: pb.debtorContry,
		},
		DebtorPacc: &PaymentAccountDTO{
			IBAN:       pb.debtorIBAN,
			AccCurency: pb.debitorCur,
		},
		Creditor: &PaymentPartyDTO{
			Name:              pb.debPartyname,
			BIC:               pb.debPartyBIC,
			Role:              "creditor",
			ContryOfResidence: pb.creditorContry,
		},
		CreditorPacc: &PaymentAccountDTO{
			IBAN:       pb.creditorIBAN,
			AccCurency: pb.debitorCur,
		},
	}
}

// BuildMany создаёт N платежей с указанными парами IBAN
func BuildMany(pairs []IBANPair) ([]*PaymentRequest, []*PaymentBuilder) {
	result := make([]*PaymentRequest, len(pairs))
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
func BuildIndependentPair() ([]*PaymentRequest, []*PaymentBuilder) {
	return BuildMany([]IBANPair{
		{Debtor: genIBAN(), Creditor: genIBAN()},
		{Debtor: genIBAN(), Creditor: genIBAN()},
	})
}

// BuildDependentChain создаёт цепочку платежей с общими IBAN
func BuildDependentChain(length int) ([]*PaymentRequest, []*PaymentBuilder) {
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
func BuildConcurrentPressure(count int) ([]*PaymentRequest, []*PaymentBuilder) {
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
func MergePaymentRequests(slices ...[]*PaymentRequest) []*PaymentRequest {
	totalLen := 0
	for _, slice := range slices {
		totalLen += len(slice)
	}

	result := make([]*PaymentRequest, 0, totalLen)
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}

func buildAndPrint(label string, requests []*PaymentRequest) {
	for i, req := range requests {
		data, err := json.MarshalIndent(req, "", "  ")
		if err != nil {
			fmt.Printf("[%s] #%d marshal error: %v\n", label, i, err)
			continue
		}
		fmt.Printf("[%s] #%d:\n%s\n\n", label, i, string(data))
	}
}

const targetURL = "http://localhost:8080/api/v1/payments"

type result struct {
	label      string
	index      int
	statusCode int
	body       string
	err        error
}

func sendPayment(label string, index int, req *PaymentRequest, results chan<- result) {
	data, err := json.Marshal(req)
	if err != nil {
		results <- result{label: label, index: index, err: fmt.Errorf("marshal: %w", err)}
		return
	}

	resp, err := http.Post(targetURL, "application/json; charset=utf-8", bytes.NewReader(data))
	if err != nil {
		results <- result{label: label, index: index, err: fmt.Errorf("http post: %w", err)}
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	results <- result{
		label:      label,
		index:      index,
		statusCode: resp.StatusCode,
		body:       string(respBody),
	}
}

type job struct {
	label   string
	buildFn func() ([]*PaymentRequest, []*PaymentBuilder)
}

func main() {
	jobs := []job{
		{
			label:   "IndependentPair",
			buildFn: BuildIndependentPair,
		},
		{
			label: "DependentChain(4)",
			buildFn: func() ([]*PaymentRequest, []*PaymentBuilder) {
				return BuildDependentChain(4)
			},
		},
		{
			label: "ConcurrentPressure(6)",
			buildFn: func() ([]*PaymentRequest, []*PaymentBuilder) {
				return BuildConcurrentPressure(6)
			},
		},
	}

	// Собираем все платежи из всех конфигураций
	type taggedRequest struct {
		label string
		index int
		req   *PaymentRequest
	}

	var allRequests []taggedRequest
	for _, j := range jobs {
		reqs, _ := j.buildFn()
		for i, req := range reqs {
			allRequests = append(allRequests, taggedRequest{label: j.label, index: i, req: req})
		}
	}

	// Отправляем все параллельно
	resultsCh := make(chan result, len(allRequests))
	var wg sync.WaitGroup

	for _, tr := range allRequests {
		wg.Add(1)
		go func(tr taggedRequest) {
			defer wg.Done()
			sendPayment(tr.label, tr.index, tr.req, resultsCh)
		}(tr)
	}

	wg.Wait()
	close(resultsCh)

	// ← вот это отсутствует в твоём коде
	for res := range resultsCh {
		if res.err != nil {
			fmt.Printf("[%s] #%d ERROR: %v\n", res.label, res.index, res.err)
			continue
		}
		fmt.Printf("[%s] #%d → HTTP %d\n%s\n\n", res.label, res.index, res.statusCode, res.body)
	}
}
