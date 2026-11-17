package order

import "testing"

func TestValidateTrimsAndAccepts(t *testing.T) {
	o := Order{Customer: "  Sara  ", Items: []Item{{Name: " Biryani ", Qty: 1, Price: 9}}}
	if err := o.Validate(); err != nil {
		t.Fatal(err)
	}
	if o.Customer != "Sara" || o.Items[0].Name != "Biryani" {
		t.Fatalf("expected trimmed values, got %+v", o)
	}
}

func TestValidateRejects(t *testing.T) {
	bad := []Order{
		{},
		{Customer: "A"},
		{Customer: "A", Items: []Item{{Name: "", Qty: 1}}},
		{Customer: "A", Items: []Item{{Name: "X", Qty: 0}}},
		{Customer: "A", Items: []Item{{Name: "X", Qty: 1, Price: -0.5}}},
	}
	for i, o := range bad {
		if err := o.Validate(); err == nil {
			t.Errorf("case %d should be rejected", i)
		}
	}
}
