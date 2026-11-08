package v2ray

import (
	"testing"
)

func TestFeedAutoRegistersProductOnSubscribe(t *testing.T) {
	feed := NewSubscriptions(10)
	// Unregistered product should auto-register and return a valid box instead of nil
	box := feed.SubscribeMessage("unregistered_product")
	if box == nil {
		t.Fatal("SubscribeMessage for unregistered product returned nil, expected auto-registration")
	}
	defer box.Cancel()

	// Should be able to receive messages on the box
	num := feed.ProductMessage("unregistered_product", "test_message")
	if num != 1 {
		t.Fatalf("ProductMessage returned %d consumers, want 1", num)
	}

	select {
	case msg := <-box.Messages:
		if msg.Body != "test_message" {
			t.Fatalf("received message body = %v, want %q", msg.Body, "test_message")
		}
	default:
		t.Fatal("box.Messages had no message")
	}
}

func TestApiFeedRegistersLiveFlowProduct(t *testing.T) {
	if ApiFeed == nil {
		t.Fatal("ApiFeed is nil")
	}
	box := ApiFeed.SubscribeMessage(LiveFlowProduct)
	if box == nil {
		t.Fatal("ApiFeed.SubscribeMessage(LiveFlowProduct) returned nil")
	}
	defer box.Cancel()
}
