package dispatcher

import (
	"context"
	"errors"
	"testing"
)

func TestNoopControllerSendMessageReturnsErrNotSteerable(t *testing.T) {
	ctrl := noopController{}
	err := ctrl.SendMessage(context.Background(), "hello")
	if !errors.Is(err, ErrNotSteerable) {
		t.Fatalf("SendMessage error = %v, want ErrNotSteerable", err)
	}
}

func TestNoopControllerInterruptReturnsErrNotSteerable(t *testing.T) {
	ctrl := noopController{}
	err := ctrl.Interrupt(context.Background())
	if !errors.Is(err, ErrNotSteerable) {
		t.Fatalf("Interrupt error = %v, want ErrNotSteerable", err)
	}
}

func TestNoopControllerSteerableReturnsFalse(t *testing.T) {
	ctrl := noopController{}
	if ctrl.Steerable() {
		t.Fatal("Steerable() = true, want false")
	}
}
