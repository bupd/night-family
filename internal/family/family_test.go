package family

import (
	"errors"
	"testing"
)

func TestLoadDefaultsRosterLoads(t *testing.T) {
	members, err := LoadDefaults()
	if err != nil {
		t.Fatalf("LoadDefaults: %v", err)
	}
	want := map[string]bool{
		"rick": false, "morty": false, "summer": false, "beth": false,
		"jerry": false, "birdperson": false, "squanchy": false,
	}
	for _, m := range members {
		if _, ok := want[m.Name]; ok {
			want[m.Name] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("default roster missing %q", name)
		}
	}
	// Defaults must be internally consistent.
	for _, m := range members {
		if issues := Validate(m); len(issues) > 0 {
			t.Errorf("default member %q invalid: %v", m.Name, issues)
		}
	}
}

func TestApplyDefaultsFillsMissingFields(t *testing.T) {
	m := Member{
		Name:         "test",
		Role:         "r",
		SystemPrompt: "p",
		Duties:       []Duty{{Type: "lint-fix", Interval: "24h"}},
	}
	m.applyDefaults()
	if m.RiskTolerance != RiskMedium {
		t.Errorf("risk_tolerance = %q, want medium", m.RiskTolerance)
	}
	if m.CostTier != CostMedium {
		t.Errorf("cost_tier = %q, want medium", m.CostTier)
	}
	if m.MaxPRsPerNight != 2 {
		t.Errorf("max_prs_per_night = %d, want 2", m.MaxPRsPerNight)
	}
	if m.Duties[0].Priority != PriorityMedium {
		t.Errorf("duty priority = %q, want medium", m.Duties[0].Priority)
	}
}

func TestValidateCatchesBadFields(t *testing.T) {
	cases := map[string]Member{
		"empty name":         {Role: "r", SystemPrompt: "p"},
		"bad name":           {Name: "BAD Name", Role: "r", SystemPrompt: "p"},
		"empty role":         {Name: "ok", SystemPrompt: "p"},
		"empty prompt":       {Name: "ok", Role: "r"},
		"bad risk":           {Name: "ok", Role: "r", SystemPrompt: "p", RiskTolerance: "banana"},
		"bad cost":           {Name: "ok", Role: "r", SystemPrompt: "p", RiskTolerance: RiskLow, CostTier: "banana"},
		"bad duty interval":  {Name: "ok", Role: "r", SystemPrompt: "p", RiskTolerance: RiskLow, CostTier: CostLow, Duties: []Duty{{Type: "lint", Interval: "forever"}}},
		"missing duty type":  {Name: "ok", Role: "r", SystemPrompt: "p", RiskTolerance: RiskLow, CostTier: CostLow, Duties: []Duty{{Interval: "24h", Priority: PriorityLow}}},
		"too many PRs/night": {Name: "ok", Role: "r", SystemPrompt: "p", RiskTolerance: RiskLow, CostTier: CostLow, MaxPRsPerNight: 999},
	}
	for name, m := range cases {
		t.Run(name, func(t *testing.T) {
			if issues := Validate(m); len(issues) == 0 {
				t.Fatalf("validation passed; want issues")
			}
		})
	}
}

func TestStoreAddGetRemove(t *testing.T) {
	s := NewStore()
	m := Member{Name: "mr-meeseeks", Role: "r", SystemPrompt: "p"}
	got, err := s.Add(m)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Fatalf("timestamps not set")
	}
	if s.Len() != 1 {
		t.Fatalf("Len = %d, want 1", s.Len())
	}

	_, err = s.Add(m)
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("second Add err = %v, want ErrDuplicate", err)
	}

	fetched, err := s.Get("mr-meeseeks")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fetched.Name != "mr-meeseeks" {
		t.Fatalf("Get.Name = %q", fetched.Name)
	}

	if _, err := s.Get("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(missing) err = %v, want ErrNotFound", err)
	}

	if err := s.Remove("mr-meeseeks"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if s.Len() != 0 {
		t.Fatalf("Len after Remove = %d, want 0", s.Len())
	}
	if err := s.Remove("mr-meeseeks"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("double Remove err = %v, want ErrNotFound", err)
	}
}

func TestStorePutUpdatesTimestamp(t *testing.T) {
	s := NewStore()
	m := Member{Name: "rick", Role: "r", SystemPrompt: "p"}
	first, err := s.Put(m)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	second, err := s.Put(Member{Name: "rick", Role: "r2", SystemPrompt: "p2"})
	if err != nil {
		t.Fatalf("Put (update): %v", err)
	}
	if !second.CreatedAt.Equal(first.CreatedAt) {
		t.Errorf("CreatedAt changed on update")
	}
	if !second.UpdatedAt.After(first.UpdatedAt) && !second.UpdatedAt.Equal(first.UpdatedAt) {
		t.Errorf("UpdatedAt did not advance")
	}
	if second.Role != "r2" {
		t.Errorf("Put did not replace role")
	}
}

func TestStoreAddRejectsInvalid(t *testing.T) {
	s := NewStore()
	_, err := s.Add(Member{Name: "BAD"}) // bad name + missing fields
	if err == nil {
		t.Fatalf("Add accepted invalid member")
	}
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %T, want ValidationError", err)
	}
	if len(ve) == 0 {
		t.Fatalf("empty issue list")
	}
}

func TestSeedReplaces(t *testing.T) {
	s := NewStore()
	s.Seed([]Member{{Name: "rick", Role: "r", SystemPrompt: "p"}})
	if s.Len() != 1 {
		t.Fatalf("Len = %d, want 1", s.Len())
	}
	s.Seed([]Member{
		{Name: "morty", Role: "r", SystemPrompt: "p"},
		{Name: "summer", Role: "r", SystemPrompt: "p"},
	})
	if s.Len() != 2 {
		t.Fatalf("Len after reseed = %d, want 2", s.Len())
	}
	if _, err := s.Get("rick"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(rick) after reseed err = %v, want ErrNotFound", err)
	}
}
