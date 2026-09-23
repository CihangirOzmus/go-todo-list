package models

import "testing"

func TestRoleValid(t *testing.T) {
	cases := []struct {
		role Role
		want bool
	}{
		{RoleUser, true},
		{RolePowerUser, true},
		{RoleAdmin, true},
		{Role(""), false},
		{Role("wizard"), false},
		{Role("USER"), false},       // the DB CHECK constraint is case-sensitive
		{Role("power user"), false}, // underscore, not a space
		{Role("admin "), false},
	}
	for _, tc := range cases {
		if got := tc.role.Valid(); got != tc.want {
			t.Errorf("Role(%q).Valid() = %v, want %v", string(tc.role), got, tc.want)
		}
	}
}

// The constants must match the values stored in Postgres, since the schema
// pins them with a CHECK constraint and the repos cast raw strings back.
func TestRoleConstantValues(t *testing.T) {
	cases := map[Role]string{
		RoleUser:      "user",
		RolePowerUser: "power_user",
		RoleAdmin:     "admin",
	}
	for role, want := range cases {
		if string(role) != want {
			t.Errorf("got %q, want %q", string(role), want)
		}
	}
}
