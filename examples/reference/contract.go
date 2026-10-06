// Package reference показывает выбранные доменные правила Go DDD-эталона с нейтральными именами.
package reference

import "errors"

// State представляет состояние ресурса.
type State string

const (
	// Initialized означает создание ресурса до публикации.
	Initialized State = "init"
	// Published означает доступность ресурса для резервирования.
	Published State = "published"
	// Reserved означает выполненное резервирование.
	Reserved State = "reserved"
)

var (
	// ErrAlreadyPublished означает повторную публикацию.
	ErrAlreadyPublished = errors.New("already published")
	// ErrUnpublished означает попытку резервирования начального состояния.
	ErrUnpublished = errors.New("resource not published")
	// ErrAlreadyReserved означает повторное резервирование.
	ErrAlreadyReserved = errors.New("already reserved")
	// ErrAlreadyProcessing означает повторную обработку запроса.
	ErrAlreadyProcessing = errors.New("already processing")
)

// Item представляет минимальную проекцию ресурса без идентификаторов и инфраструктуры.
type Item struct {
	state     State
	image     string
	reference string
}

// NewItem создаёт начальное состояние ресурса.
func NewItem() *Item {
	return &Item{state: Initialized, image: "", reference: ""}
}

// State возвращает подтверждённое состояние ресурса.
func (i *Item) State() State {
	return i.state
}

// Publish публикует ресурс, отвергая повторную публикацию.
func (i *Item) Publish() error {
	if i.state == Published {
		return ErrAlreadyPublished
	}

	i.state = Published

	return nil
}

// Reserve резервирует опубликованный ресурс и сохраняет состояние при отказе.
func (i *Item) Reserve() error {
	if i.state == Reserved {
		return ErrAlreadyReserved
	}

	if i.state != Published {
		return ErrUnpublished
	}

	i.state = Reserved

	return nil
}

// SetImage меняет ключ вложения и очищает ранее подтверждённую ссылку.
func (i *Item) SetImage(key string) {
	i.image = key
	i.reference = ""
}

// ConfirmImage сохраняет непрозрачную ссылку на подтверждённое вложение.
func (i *Item) ConfirmImage(value string) {
	i.reference = value
}

// ImageReference возвращает ключ и подтверждённую ссылку вложения.
func (i *Item) ImageReference() (string, string) {
	return i.image, i.reference
}

// Request представляет минимальную проекцию created/processing.
type Request struct {
	processing bool
}

// NewRequest создаёт ещё не обработанный запрос.
func NewRequest() *Request {
	return &Request{processing: false}
}

// Process переводит запрос в processing, сохраняя состояние при повторном вызове.
func (r *Request) Process() error {
	if r.processing {
		return ErrAlreadyProcessing
	}

	r.processing = true

	return nil
}

// Processing возвращает признак обработки запроса.
func (r *Request) Processing() bool {
	return r.processing
}
