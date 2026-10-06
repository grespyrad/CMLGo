package reference_test

import (
	"errors"
	"testing"

	"github.com/grespyrad/CMLGo/examples/reference"
)

// TestReservationGuard проверяет ошибочные переходы и сохранение состояния при отказе.
func TestReservationGuard(t *testing.T) {
	t.Parallel()

	item := reference.NewItem()
	assertStateResult(t, item, item.Reserve(), reference.ErrUnpublished, reference.Initialized)
	assertStateResult(t, item, item.Publish(), nil, reference.Published)
	assertStateResult(t, item, item.Reserve(), nil, reference.Reserved)
	assertStateResult(t, item, item.Reserve(), reference.ErrAlreadyReserved, reference.Reserved)
	assertStateResult(t, item, item.Publish(), nil, reference.Published)
	assertStateResult(t, item, item.Publish(), reference.ErrAlreadyPublished, reference.Published)
}

// TestImageReferenceReset проверяет смену вложения без сохранения устаревшей ссылки.
func TestImageReferenceReset(t *testing.T) {
	t.Parallel()

	item := reference.NewItem()
	item.SetImage("asset-1")
	item.ConfirmImage("confirmed-1")
	item.SetImage("asset-2")

	key, confirmed := item.ImageReference()
	if key != "asset-2" || confirmed != "" {
		t.Fatalf("%q %q", key, confirmed)
	}
}

// TestProcessingGuard проверяет повторную обработку с сохранением processing.
func TestProcessingGuard(t *testing.T) {
	t.Parallel()

	request := reference.NewRequest()
	if request.Processing() {
		t.Fatal("new request is processing")
	}

	if err := request.Process(); err != nil || !request.Processing() {
		t.Fatalf("first processing: %v", err)
	}

	if err := request.Process(); !errors.Is(err, reference.ErrAlreadyProcessing) || !request.Processing() {
		t.Fatalf("repeat processing: %v", err)
	}
}

func assertStateResult(t *testing.T, item *reference.Item, err, want error, state reference.State) {
	t.Helper()

	if !errors.Is(err, want) || item.State() != state {
		t.Fatalf("result %v, state %s; want %v, %s", err, item.State(), want, state)
	}
}
