package validator_test

import (
	"reflect"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestResolverValidationRefusesService(t *testing.T) {
	validate := validator.New()
	for _, test := range []struct{ tag, value string }{
		{"tcp4_addr", "127.0.0.1:1234"},
		{"tcp6_addr", "[::1]:1234"},
		{"tcp_addr", "127.0.0.1:1234"},
		{"udp4_addr", "127.0.0.1:1234"},
		{"udp6_addr", "[::1]:1234"},
		{"udp_addr", "127.0.0.1:1234"},
		{"ip4_addr", "127.0.0.1"},
		{"ip6_addr", "::1"},
		{"ip_addr", "127.0.0.1"},
		{"unix_addr", "/tmp/gomad-adapter.sock"},
	} {
		t.Run(test.tag, func(t *testing.T) {
			defer func() {
				if got := recover(); got != "gomad: Validator address resolution is unsupported" {
					t.Fatalf("resolver validation panic = %v, want explicit unsupported service", got)
				}
			}()
			if err := validate.Var(test.value, test.tag); err != nil {
				t.Fatalf("literal resolver fixture was rejected before refusal: %v", err)
			}
		})
	}
}

func TestOrdinaryValidation(t *testing.T) {
	type request struct {
		Name      string   `validate:"required,min=3,max=12"`
		Email     string   `validate:"email"`
		Mode      string   `validate:"oneof=read write"`
		Addresses []string `validate:"required,dive,ip"`
		IPv4      string   `validate:"ipv4"`
		IPv6      string   `validate:"ipv6"`
	}
	validate := validator.New()
	valid := request{"sample", "sample@example.com", "write", []string{"127.0.0.1", "::1"}, "127.0.0.1", "::1"}
	if err := validate.Struct(valid); err != nil {
		t.Fatal(err)
	}
	invalid := request{"x", "broken", "unknown", []string{"not-an-ip"}, "::1", "127.0.0.1"}
	err := validate.Struct(invalid)
	errors, ok := err.(validator.ValidationErrors)
	if !ok {
		t.Fatalf("invalid fields = %T %v", err, err)
	}
	var got []string
	for _, field := range errors {
		got = append(got, field.Field()+":"+field.Tag())
	}
	want := []string{"Name:min", "Email:email", "Mode:oneof", "Addresses[0]:ip", "IPv4:ipv4", "IPv6:ipv6"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid fields = %v, want %v", got, want)
	}
}
