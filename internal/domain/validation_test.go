package domain

import "testing"

func TestValidateColor_ValidColors(t *testing.T) {
	valid := []string{"red", "orange", "yellow", "green", "teal", "blue", "purple", "pink"}
	for _, color := range valid {
		if err := ValidateColor(color); err != nil {
			t.Errorf("ValidateColor(%q) returned error: %v", color, err)
		}
	}
}

func TestValidateColor_InvalidColors(t *testing.T) {
	invalid := []string{"", "black", "white", "magenta", "RED", "Blue", " red", "red "}
	for _, color := range invalid {
		if err := ValidateColor(color); err == nil {
			t.Errorf("ValidateColor(%q) should have returned error", color)
		}
	}
}

func TestValidateEmail_ValidEmails(t *testing.T) {
	valid := []string{
		"user@example.com",
		"alice@test.org",
		"name.surname@domain.co",
		"user+tag@example.com",
	}
	for _, email := range valid {
		if err := ValidateEmail(email); err != nil {
			t.Errorf("ValidateEmail(%q) returned error: %v", email, err)
		}
	}
}

func TestValidateEmail_InvalidEmails(t *testing.T) {
	invalid := []string{
		"",
		"notanemail",
		"@missing-local.com",
		"missing-domain@",
		"spaces in@email.com",
	}
	for _, email := range invalid {
		if err := ValidateEmail(email); err == nil {
			t.Errorf("ValidateEmail(%q) should have returned error", email)
		}
	}
}

func TestValidateEmail_RejectsDisplayName(t *testing.T) {
	invalid := []string{
		`"John Doe" <john@example.com>`,
		`John <john@example.com>`,
		`<john@example.com>`,
	}
	for _, email := range invalid {
		if err := ValidateEmail(email); err == nil {
			t.Errorf("ValidateEmail(%q) should reject display-name format", email)
		}
	}
}

func TestValidateEditScope_ValidValues(t *testing.T) {
	for _, v := range []string{"single", "future", "all"} {
		if err := ValidateEditScope(v); err != nil {
			t.Errorf("ValidateEditScope(%q) returned error: %v", v, err)
		}
	}
}

func TestValidateEditScope_InvalidValues(t *testing.T) {
	for _, v := range []string{"", "none", "every", "SINGLE", "Single", "bogus"} {
		if err := ValidateEditScope(v); err == nil {
			t.Errorf("ValidateEditScope(%q) should return error", v)
		}
	}
}

func TestValidateDeleteScope_ValidValues(t *testing.T) {
	for _, v := range []string{"single", "future", "all"} {
		if err := ValidateDeleteScope(v); err != nil {
			t.Errorf("ValidateDeleteScope(%q) returned error: %v", v, err)
		}
	}
}

func TestValidateDeleteScope_InvalidValues(t *testing.T) {
	for _, v := range []string{"", "none", "every", "SINGLE", "bogus"} {
		if err := ValidateDeleteScope(v); err == nil {
			t.Errorf("ValidateDeleteScope(%q) should return error", v)
		}
	}
}

func TestValidateRepeatType_ValidValues(t *testing.T) {
	for _, v := range []string{"", "daily", "weekly", "biweekly", "monthly"} {
		if err := ValidateRepeatType(v); err != nil {
			t.Errorf("ValidateRepeatType(%q) returned error: %v", v, err)
		}
	}
}

func TestValidateRepeatType_InvalidValues(t *testing.T) {
	for _, v := range []string{"yearly", "hourly", "DAILY", "Weekly", "bogus"} {
		if err := ValidateRepeatType(v); err == nil {
			t.Errorf("ValidateRepeatType(%q) should return error", v)
		}
	}
}

func TestValidateStatusFilter_ValidValues(t *testing.T) {
	for _, v := range []string{"", "under", "good", "over"} {
		if err := ValidateStatusFilter(v); err != nil {
			t.Errorf("ValidateStatusFilter(%q) returned error: %v", v, err)
		}
	}
}

func TestValidateStatusFilter_InvalidValues(t *testing.T) {
	for _, v := range []string{"bad", "full", "UNDER", "pending", "bogus"} {
		if err := ValidateStatusFilter(v); err == nil {
			t.Errorf("ValidateStatusFilter(%q) should return error", v)
		}
	}
}

func TestValidateRoleFilter_ValidValues(t *testing.T) {
	for _, v := range []string{"", "participants", "organizers", "all"} {
		if err := ValidateRoleFilter(v); err != nil {
			t.Errorf("ValidateRoleFilter(%q) returned error: %v", v, err)
		}
	}
}

func TestValidateRoleFilter_InvalidValues(t *testing.T) {
	for _, v := range []string{"admin", "admins", "PARTICIPANTS", "users", "bogus"} {
		if err := ValidateRoleFilter(v); err == nil {
			t.Errorf("ValidateRoleFilter(%q) should return error", v)
		}
	}
}
