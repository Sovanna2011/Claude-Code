package domain

import "fmt"

// PlanAction is a workflow command applied to a planning version.
type PlanAction string

const (
	ActionSubmit    PlanAction = "SUBMIT"
	ActionRecall    PlanAction = "RECALL"
	ActionApprove   PlanAction = "APPROVE"
	ActionReject    PlanAction = "REJECT"
	ActionRelease   PlanAction = "RELEASE"
	ActionSupersede PlanAction = "SUPERSEDE"
	ActionClose     PlanAction = "CLOSE"
	ActionReopen    PlanAction = "REOPEN"
)

// Transition is one edge of the plan version state machine.
type Transition struct {
	Action PlanAction
	From   PlanStatus
	To     PlanStatus
	// Permission the caller must hold. Permissions are checked in the service
	// layer against the caller's roles; the table keeps the requirement next to
	// the transition it guards so the two cannot drift apart.
	Permission string
	// ReasonRequired forces the caller to supply a reason that is written to
	// the audit trail. Reopening a released plan always requires one.
	ReasonRequired bool
}

// planTransitions is the complete state machine from section 4. Anything not
// listed here is rejected.
var planTransitions = []Transition{
	{ActionSubmit, StatusDraft, StatusInReview, PermPlanSubmit, false},
	{ActionSubmit, StatusRejected, StatusInReview, PermPlanSubmit, false},
	{ActionRecall, StatusInReview, StatusDraft, PermPlanSubmit, false},
	{ActionApprove, StatusInReview, StatusApproved, PermPlanApprove, false},
	{ActionReject, StatusInReview, StatusRejected, PermPlanApprove, true},
	{ActionRelease, StatusApproved, StatusReleased, PermPlanRelease, false},
	{ActionSupersede, StatusReleased, StatusSuperseded, PermPlanRelease, true},
	{ActionClose, StatusReleased, StatusClosed, PermPlanRelease, true},
	{ActionClose, StatusSuperseded, StatusClosed, PermPlanRelease, true},
	{ActionReopen, StatusApproved, StatusDraft, PermPlanReopen, true},
	{ActionReopen, StatusReleased, StatusDraft, PermPlanReopen, true},
	{ActionReopen, StatusClosed, StatusDraft, PermPlanReopen, true},
}

// LookupTransition finds the transition for an action in a status.
func LookupTransition(from PlanStatus, action PlanAction) (Transition, error) {
	for _, t := range planTransitions {
		if t.From == from && t.Action == action {
			return t, nil
		}
	}
	return Transition{}, fmt.Errorf("%w: cannot %s a plan in status %s", ErrStateTransition, action, from)
}

// AllowedActions lists the actions available from a status, for the UI to
// enable or disable buttons without hard-coding the rules a second time.
func AllowedActions(from PlanStatus, planType PlanType) []PlanAction {
	var out []PlanAction
	for _, t := range planTransitions {
		if t.From != from {
			continue
		}
		if !actionAllowedForType(t.Action, planType) {
			continue
		}
		out = append(out, t.Action)
	}
	return out
}

// actionAllowedForType applies the rules that depend on what kind of version it
// is rather than on its status.
func actionAllowedForType(action PlanAction, planType PlanType) bool {
	switch planType {
	case PlanTypeWhatIf:
		// A simulation is never submitted, approved or released: it exists to
		// be compared, and is copied into a revised plan if it is adopted.
		return false
	case PlanTypeActual:
		// The actuals version is a container for recorded facts. It is closed
		// with the season, never approved or released.
		return action == ActionClose || action == ActionReopen
	default:
		return true
	}
}

// ApplyTransition validates and returns the next status.
func ApplyTransition(v PlanVersion, action PlanAction, reason string) (Transition, error) {
	if !actionAllowedForType(action, v.PlanType) {
		return Transition{}, fmt.Errorf("%w: %s is not available for a %s version",
			ErrStateTransition, action, v.PlanType)
	}
	t, err := LookupTransition(v.Status, action)
	if err != nil {
		return Transition{}, err
	}
	if t.ReasonRequired && reason == "" {
		return Transition{}, fmt.Errorf("%w: %s requires a reason for the audit trail",
			ErrValidation, action)
	}
	return t, nil
}

// IsDateLocked reports whether a business date may no longer be edited in this
// version. A released plan locks everything up to and including LockedThrough;
// a closed plan locks everything.
func IsDateLocked(v PlanVersion, date BusinessDate) bool {
	switch v.Status {
	case StatusDraft, StatusRejected:
		return false
	case StatusClosed, StatusSuperseded:
		return true
	case StatusInReview, StatusApproved:
		return true
	case StatusReleased:
		if v.LockedThrough == "" {
			return true
		}
		return date <= v.LockedThrough
	default:
		return true
	}
}

// CheckWritable returns an error when a row for the given date cannot be
// written to this version.
func CheckWritable(v PlanVersion, date BusinessDate, series Series) error {
	if series == SeriesActual && v.PlanType != PlanTypeActual && v.PlanType != PlanTypeLatestEstimate {
		return fmt.Errorf("%w: actuals cannot be posted to a %s version", ErrValidation, v.PlanType)
	}
	if series == SeriesPlan && v.PlanType == PlanTypeActual {
		return fmt.Errorf("%w: plan values cannot be written to the actuals version", ErrValidation)
	}
	// Actuals are recorded against the actuals version for as long as the
	// season is open; the plan locking rules apply to planned values only.
	if series == SeriesActual {
		if v.Status == StatusClosed {
			return fmt.Errorf("%w: the period is closed for %s", ErrLocked, date)
		}
		return nil
	}
	if IsDateLocked(v, date) {
		return fmt.Errorf("%w: %s is locked in version %s (status %s)", ErrLocked, date, v.Code, v.Status)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Permissions
// ---------------------------------------------------------------------------

// Permission strings. The role to permission mapping lives in
// docs/03-module-and-authorization-matrix.md and in auth.DefaultRoles.
const (
	PermMasterDataRead  = "masterdata:read"
	PermMasterDataWrite = "masterdata:write"
	PermPlanRead        = "plan:read"
	PermPlanWrite       = "plan:write"
	PermPlanSubmit      = "plan:submit"
	PermPlanApprove     = "plan:approve"
	PermPlanRelease     = "plan:release"
	PermPlanReopen      = "plan:reopen"
	PermActualCane      = "actual:cane"
	PermActualProduce   = "actual:production"
	PermActualShip      = "actual:shipment"
	PermActualStock     = "actual:stock"
	PermQualityWrite    = "quality:write"
	PermQualityRelease  = "quality:release"
	PermDowntimeWrite   = "downtime:write"
	PermMaterialsRead   = "materials:read"
	// PermCostRead and PermCostWrite guard the costing separately from the
	// operational figures. What a factory pays for cane is commercially
	// sensitive in a way that how much cane it crushed is not, so a plan reader
	// does not see it by default.
	PermCostRead         = "cost:read"
	PermCostWrite        = "cost:write"
	PermReportRead       = "report:read"
	PermAuditRead        = "audit:read"
	PermAdmin            = "admin"
	PermCapacityOverride = "capacity:override"
)
