// Package planner builds the ordered list of (member, duty) pairs a
// night should attempt, given a family roster, the duty registry, a
// schedule, and a budget ceiling.
//
// v1 is stateless and deterministic: every enabled duty on every member
// is considered "due" unless something explicitly disqualifies it (duty
// type unknown, would exceed the budget, would exceed the member's
// max_prs_per_night). The persistence layer (interval tracking from
// past runs) lands in a follow-up.
package planner

import (
	"fmt"
	"sort"
	"time"

	"github.com/bupd/night-family/internal/duty"
	"github.com/bupd/night-family/internal/family"
	"github.com/bupd/night-family/internal/schedule"
)

// Slot is one planned (member, duty) pair, in the order we intend to
// dispatch it.
type Slot struct {
	Member          string          `json:"member"`
	Duty            string          `json:"duty"`
	Priority        family.Priority `json:"priority"`
	CostTier        family.Cost     `json:"cost_tier"`
	Risk            family.Risk     `json:"risk"`
	Output          duty.Output     `json:"output"`
	EstimatedTokens int             `json:"estimated_tokens"`
	// Reason, when non-empty, is set for slots we kept but flagged
	// (e.g. "duty type not in registry").
	Reason string `json:"reason,omitempty"`
}

// Skipped records why a candidate duty was dropped from the plan. Kept
// so the UI and the API can show the full picture.
type Skipped struct {
	Member string `json:"member"`
	Duty   string `json:"duty"`
	Reason string `json:"reason"`
}

// Plan is the full result of a planning pass.
type Plan struct {
	GeneratedAt     time.Time      `json:"generated_at"`
	WindowStart     time.Time      `json:"window_start"`
	WindowEnd       time.Time      `json:"window_end"`
	BudgetTokens    int            `json:"budget_tokens"`
	ReservedTokens  int            `json:"reserved_tokens"`
	Slots           []Slot         `json:"slots"`
	Skipped         []Skipped      `json:"skipped,omitempty"`
	PerMemberCounts map[string]int `json:"per_member_counts"`
}

// Input is everything the planner needs to compute a Plan.
type Input struct {
	Family   *family.Store
	Duties   *duty.Registry
	Schedule *schedule.Schedule
	Now      time.Time
	// BudgetTokens is the upper bound we'll try to stay under. Zero
	// means "no budget tracking — keep every slot".
	BudgetTokens int
}

// TokenEstimate returns our cheap guess for how many tokens a member ×
// duty combination will consume. Deliberately coarse; the real number
// comes from the provider once a duty actually runs.
func TokenEstimate(memberCost family.Cost, dutyCost family.Cost) int {
	base := 4000
	switch memberCost {
	case family.CostMedium:
		base = 8000
	case family.CostHigh:
		base = 16000
	}
	switch dutyCost {
	case family.CostMedium:
		base += 4000
	case family.CostHigh:
		base += 12000
	}
	return base
}

// priorityRank maps a priority to an ordering value (higher = runs first).
func priorityRank(p family.Priority) int {
	switch p {
	case family.PriorityHigh:
		return 2
	case family.PriorityMedium:
		return 1
	case family.PriorityLow:
		return 0
	}
	return 1
}

// Plan computes the next night's plan from Input.
func (in Input) Plan() (Plan, error) {
	if in.Family == nil {
		return Plan{}, fmt.Errorf("planner: Input.Family required")
	}
	if in.Duties == nil {
		return Plan{}, fmt.Errorf("planner: Input.Duties required")
	}
	if in.Schedule == nil {
		return Plan{}, fmt.Errorf("planner: Input.Schedule required")
	}
	if in.Now.IsZero() {
		in.Now = time.Now()
	}

	winStart, winEnd, err := in.Schedule.Next(in.Now)
	if err != nil {
		return Plan{}, fmt.Errorf("planner: resolve window: %w", err)
	}

	var slots []Slot
	var skipped []Skipped
	perMember := make(map[string]int)

	members := in.Family.List()
	for _, m := range members {
		for _, d := range m.Duties {
			info, known := in.Duties.Get(d.Type)
			reason := ""
			if !known {
				// Keep it as a prompt-only duty with best-guess metadata,
				// but record the reason so callers can surface a warning.
				reason = "duty type not in built-in registry (treated as prompt-only)"
				info = duty.Info{
					Type:     d.Type,
					Output:   duty.OutputNote,
					CostTier: family.CostMedium,
					Risk:     family.RiskMedium,
				}
			}
			cap := m.MaxPRsPerNight
			if cap > 0 && perMember[m.Name] >= cap {
				skipped = append(skipped, Skipped{
					Member: m.Name, Duty: d.Type,
					Reason: fmt.Sprintf("member cap (max_prs_per_night=%d) reached", cap),
				})
				continue
			}
			slot := Slot{
				Member:          m.Name,
				Duty:            d.Type,
				Priority:        d.Priority,
				CostTier:        info.CostTier,
				Risk:            info.Risk,
				Output:          info.Output,
				EstimatedTokens: TokenEstimate(m.CostTier, info.CostTier),
				Reason:          reason,
			}
			slots = append(slots, slot)
			perMember[m.Name]++
		}
	}

	// Order slots by priority desc, then member name, then duty name.
	sort.SliceStable(slots, func(i, j int) bool {
		if pi, pj := priorityRank(slots[i].Priority), priorityRank(slots[j].Priority); pi != pj {
			return pi > pj
		}
		if slots[i].Member != slots[j].Member {
			return slots[i].Member < slots[j].Member
		}
		return slots[i].Duty < slots[j].Duty
	})

	// Budget pass: drop from the tail until we fit.
	reserved := 0
	if in.BudgetTokens > 0 {
		kept := slots[:0]
		for _, s := range slots {
			if reserved+s.EstimatedTokens > in.BudgetTokens {
				skipped = append(skipped, Skipped{
					Member: s.Member, Duty: s.Duty,
					Reason: fmt.Sprintf(
						"would exceed budget (reserved=%d, slot=%d, budget=%d)",
						reserved, s.EstimatedTokens, in.BudgetTokens),
				})
				// Update per-member counts to reflect the drop.
				if perMember[s.Member] > 0 {
					perMember[s.Member]--
				}
				continue
			}
			reserved += s.EstimatedTokens
			kept = append(kept, s)
		}
		slots = kept
	} else {
		for _, s := range slots {
			reserved += s.EstimatedTokens
		}
	}

	return Plan{
		GeneratedAt:     in.Now.UTC(),
		WindowStart:     winStart,
		WindowEnd:       winEnd,
		BudgetTokens:    in.BudgetTokens,
		ReservedTokens:  reserved,
		Slots:           slots,
		Skipped:         skipped,
		PerMemberCounts: perMember,
	}, nil
}
