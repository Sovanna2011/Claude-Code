package domain

import (
	"fmt"
	"time"
)

// A planting projection is the committed answer to "what will be planted, where, when and with
// which variety". Everything downstream — the activity plan, the machinery schedule, the material
// requirement — is generated from an approved one, which is why it carries a workflow rather than
// being ordinary master data.

const (
	ProjectionDraft       = "Draft"
	ProjectionSubmitted   = "Submitted"
	ProjectionUnderReview = "UnderReview"
	ProjectionApproved    = "Approved"
	ProjectionRejected    = "Rejected"
	ProjectionRevised     = "Revised"
	ProjectionClosed      = "Closed"
)

var ValidProjectionStatuses = []string{
	ProjectionDraft, ProjectionSubmitted, ProjectionUnderReview,
	ProjectionApproved, ProjectionRejected, ProjectionRevised, ProjectionClosed}

// The workflow actions, as they appear in the request path: POST /api/projections/{id}/submit.
const (
	ActionSubmit              = "Submit"
	ActionReview              = "Review"
	ActionApprove             = "Approve"
	ActionReject              = "Reject"
	ActionReturnForCorrection = "ReturnForCorrection"
	ActionRevise              = "Revise"
	ActionClose               = "Close"
)

// Transition is one edge of the workflow: who may take this action, from which statuses, and where
// it lands. Keeping the graph as data rather than a switch means the UI can be told what is
// possible next from the same source the server enforces.
type Transition struct {
	Action string   `json:"action"`
	From   []string `json:"from"`
	To     string   `json:"to"`
	Roles  []string `json:"roles"`

	// NeedsReason makes the comment mandatory. Refusing a plan or reopening an approved one
	// without saying why leaves the next planner guessing.
	NeedsReason bool `json:"needsReason"`
}

// ProjectionWorkflow is the whole graph. A planner prepares and submits; a manager reviews and
// decides; either may open a revision of a plan that has been decided.
var ProjectionWorkflow = []Transition{
	{Action: ActionSubmit, From: []string{ProjectionDraft}, To: ProjectionSubmitted,
		Roles: []string{RoleAdmin, RoleManager, RolePlanner}},
	{Action: ActionReview, From: []string{ProjectionSubmitted}, To: ProjectionUnderReview,
		Roles: []string{RoleAdmin, RoleManager}},
	{Action: ActionApprove, From: []string{ProjectionSubmitted, ProjectionUnderReview}, To: ProjectionApproved,
		Roles: []string{RoleAdmin, RoleManager}},
	{Action: ActionReject, From: []string{ProjectionSubmitted, ProjectionUnderReview}, To: ProjectionRejected,
		Roles: []string{RoleAdmin, RoleManager}, NeedsReason: true},
	{Action: ActionReturnForCorrection, From: []string{ProjectionSubmitted, ProjectionUnderReview}, To: ProjectionDraft,
		Roles: []string{RoleAdmin, RoleManager}, NeedsReason: true},
	{Action: ActionRevise, From: []string{ProjectionApproved, ProjectionRejected}, To: ProjectionRevised,
		Roles: []string{RoleAdmin, RoleManager, RolePlanner}, NeedsReason: true},
	{Action: ActionClose, From: []string{ProjectionApproved}, To: ProjectionClosed,
		Roles: []string{RoleAdmin, RoleManager}},
}

// FindTransition looks an action up by name, whatever case the caller spelled it in.
func FindTransition(action string) (Transition, bool) {
	for _, t := range ProjectionWorkflow {
		if IsOneOf(action, []string{t.Action}) {
			return t, true
		}
	}
	return Transition{}, false
}

// AllowedActions is what this caller may do to a projection in this status — the list the UI turns
// into buttons. It is derived from the same graph the server enforces, so a button that appears
// always works and one that would be refused never appears.
func AllowedActions(status string, u User) []string {
	out := []string{}
	for _, t := range ProjectionWorkflow {
		if IsOneOf(status, t.From) && u.HasAnyRole(t.Roles...) {
			out = append(out, t.Action)
		}
	}
	return out
}

// CheckTransition is the whole guard: the action exists, this status can take it, and this caller
// is allowed to. The three refusals are deliberately different — 400 for a nonsense action, 409
// for the right action at the wrong time, 403 for the wrong person.
func CheckTransition(action, status string, u User) (Transition, error) {
	t, ok := FindTransition(action)
	if !ok {
		return t, BadRequest("UNKNOWN_ACTION", fmt.Sprintf("There is no workflow action called %q.", action))
	}
	if !IsOneOf(status, t.From) {
		return t, Conflict("INVALID_TRANSITION", fmt.Sprintf(
			"A projection that is %s cannot be %sd; that is only possible from: %s.",
			status, t.Action, joinWords(t.From)))
	}
	if !u.HasAnyRole(t.Roles...) {
		return t, Forbidden(fmt.Sprintf(
			"Your role (%s) may not %s a projection; that needs one of: %s.",
			u.Role, t.Action, joinWords(t.Roles)))
	}
	return t, nil
}

// LinesEditable reports whether the lines may still be changed. An approved plan is a commitment
// that other modules have already read; correcting one means opening a revision, not editing it
// behind their backs.
func LinesEditable(status string) bool { return status == ProjectionDraft }

func joinWords(values []string) string {
	switch len(values) {
	case 0:
		return "nothing"
	case 1:
		return values[0]
	}
	out := ""
	for i, v := range values {
		switch {
		case i == 0:
			out = v
		case i == len(values)-1:
			out += " or " + v
		default:
			out += ", " + v
		}
	}
	return out
}

// ---------------------------------------------------------------- the records

// Projection is the header. The four totals are derived from the lines by the database and have no
// setter here: the JSON is read-only for them, and ProjectionInput has no field to send one.
type Projection struct {
	ID             int    `json:"id"`
	CompanyID      int    `json:"companyId"`
	CompanyName    string `json:"companyName"`
	PlantationID   int    `json:"plantationId"`
	PlantationName string `json:"plantationName"`
	CropSeasonID   int    `json:"cropSeasonId"`
	CropSeasonCode string `json:"cropSeasonCode"`
	CropYear       int    `json:"cropYear"`

	ProjectionNo string `json:"projectionNo"`
	Revision     int    `json:"revision"`
	SupersedesID *int   `json:"supersedesId"`
	SupersededBy *int   `json:"supersededById"`
	IsCurrent    bool   `json:"isCurrent"`

	ProjectionDate string `json:"projectionDate"`
	PlanningStart  string `json:"planningStart"`
	PlanningEnd    string `json:"planningEnd"`

	TotalProjectedAreaHa   float64 `json:"totalProjectedAreaHa"`
	TotalHarvestableAreaHa float64 `json:"totalHarvestableAreaHa"`
	TotalExpectedTons      float64 `json:"totalExpectedTons"`
	LineCount              int     `json:"lineCount"`

	Status          string  `json:"status"`
	PreparedBy      *string `json:"preparedBy"`
	SubmittedBy     *string `json:"submittedBy"`
	SubmittedAt     *string `json:"submittedAt"`
	ReviewedBy      *string `json:"reviewedBy"`
	ReviewedAt      *string `json:"reviewedAt"`
	ApprovedBy      *string `json:"approvedBy"`
	ApprovedAt      *string `json:"approvedAt"`
	RejectedBy      *string `json:"rejectedBy"`
	RejectedAt      *string `json:"rejectedAt"`
	RejectionReason *string `json:"rejectionReason"`
	RevisionReason  *string `json:"revisionReason"`
	ClosedBy        *string `json:"closedBy"`
	ClosedAt        *string `json:"closedAt"`

	Remark    *string   `json:"remark"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy"`

	// Filled by GetProjection, left empty by the list.
	Lines   []ProjectionLine   `json:"lines,omitempty"`
	History []ProjectionAction `json:"history,omitempty"`

	// What this caller may do next. Derived, never stored.
	Actions       []string `json:"actions"`
	LinesEditable bool     `json:"linesEditable"`
}

// ProjectionInput is what a caller may set. There is no field for a total, a status, or any of the
// workflow stamps — those move only through the workflow endpoints.
type ProjectionInput struct {
	CompanyID      int     `json:"companyId"`
	PlantationID   int     `json:"plantationId"`
	CropSeasonID   int     `json:"cropSeasonId"`
	ProjectionNo   string  `json:"projectionNo"`
	ProjectionDate string  `json:"projectionDate"`
	PlanningStart  string  `json:"planningStart"`
	PlanningEnd    string  `json:"planningEnd"`
	PreparedBy     *string `json:"preparedBy"`
	Remark         *string `json:"remark"`
	Version        int     `json:"version"`
}

// ProjectionLine is one block's share of the plan.
type ProjectionLine struct {
	ID           int    `json:"id"`
	ProjectionID int    `json:"projectionId"`
	BlockID      int    `json:"blockId"`
	BlockCode    string `json:"blockCode"`
	BlockName    string `json:"blockName"`
	ZoneID       int    `json:"zoneId"`
	ZoneCode     string `json:"zoneCode"`
	FarmID       int    `json:"farmId"`
	FarmCode     string `json:"farmCode"`
	FarmName     string `json:"farmName"`

	CaneVarietyID   int    `json:"caneVarietyId"`
	CaneVarietyCode string `json:"caneVarietyCode"`
	PlantingType    string `json:"plantingType"`

	AvailableAreaHa         float64 `json:"availableAreaHa"`
	ProjectedPlantingAreaHa float64 `json:"projectedPlantingAreaHa"`

	PlannedPlantingStart string  `json:"plannedPlantingStart"`
	PlannedPlantingEnd   string  `json:"plannedPlantingEnd"`
	ExpectedHarvestDate  *string `json:"expectedHarvestDate"`

	ExpectedYieldPerHa float64 `json:"expectedYieldPerHa"`
	ExpectedLossPct    float64 `json:"expectedLossPercent"`

	HarvestableAreaHa      float64 `json:"harvestableAreaHa"`
	ExpectedProductionTons float64 `json:"expectedProductionTons"`

	// The seed cane this line needs, from the variety's seed rate. Derived on read rather than
	// stored: it is a function of the line and the variety, and storing it would let the two drift.
	RequiredSeedCaneTons float64 `json:"requiredSeedCaneTons"`

	Priority int     `json:"priority"`
	Remark   *string `json:"remark"`

	MapURL string `json:"mapUrl"`
}

// ProjectionLineInput deliberately has no available-area field. The snapshot is taken from the
// block by the server; letting a caller send it would turn the "within the block" check into a
// check against a number the caller chose.
type ProjectionLineInput struct {
	BlockID                 int      `json:"blockId"`
	CaneVarietyID           int      `json:"caneVarietyId"`
	PlantingType            string   `json:"plantingType"`
	ProjectedPlantingAreaHa float64  `json:"projectedPlantingAreaHa"`
	PlannedPlantingStart    string   `json:"plannedPlantingStart"`
	PlannedPlantingEnd      string   `json:"plannedPlantingEnd"`
	ExpectedHarvestDate     *string  `json:"expectedHarvestDate"`
	ExpectedYieldPerHa      *float64 `json:"expectedYieldPerHa"`
	ExpectedLossPercent     *float64 `json:"expectedLossPercent"`
	Priority                *int     `json:"priority"`
	Remark                  *string  `json:"remark"`
}

// ProjectionAction is one step of the trail: who moved the plan, from where to where, and why.
type ProjectionAction struct {
	ID         int64   `json:"id"`
	Action     string  `json:"action"`
	FromStatus string  `json:"fromStatus"`
	ToStatus   string  `json:"toStatus"`
	Actor      string  `json:"actor"`
	At         string  `json:"at"`
	Comments   *string `json:"comments"`
}

// WorkflowRequest is the body of every workflow endpoint.
type WorkflowRequest struct {
	Comments string `json:"comments"`
	Version  int    `json:"version"`
}
