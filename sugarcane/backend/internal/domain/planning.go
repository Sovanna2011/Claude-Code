package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Turning an approved projection into a dated programme of work.
//
// This is a pure function of its inputs: the blocks to be planted, the activity master with its
// dependency chain, and the working calendar. No database, no clock, no ordering surprises — the
// same inputs always produce the same schedule, which is what makes it testable and what lets the
// what-if module run it against a different calendar without touching the saved plan.

// PlanActivity is one step of the programme as the engine needs it.
type PlanActivity struct {
	ID         int
	Code       string
	Name       string
	Category   string
	CropType   string // NewPlanting, Ratoon or Both
	SequenceNo int

	// Days from the planting date. Negative for the work that comes first: the specification's
	// land survey sits at -45, planting itself at 0, the first ratoon care well after.
	StartDayOffset int

	CapacityPerDay  float64 // hectares a crew or machine set covers in a working day
	DurationPerHa   float64 // hours
	LabourDaysPerHa float64

	IsMandatory       bool
	RequiresTractor   bool
	RequiresEquipment bool
	RequiresMaterial  bool
	RequiresLabour    bool

	DependsOn []PlanDependency
}

// PlanDependency is "this activity waits for that one", with the lag in working days.
type PlanDependency struct {
	DependsOnID int
	LagDays     int
	IsBlocking  bool
}

// PlanLine is one block's share of an approved projection — what the programme is generated for.
type PlanLine struct {
	ProjectionLineID int
	BlockID          int
	BlockCode        string
	AreaHa           float64
	PlantingType     string
	PlantingStart    time.Time
}

// PlanTask is one activity on one block, dated.
type PlanTask struct {
	ProjectionLineID int    `json:"projectionLineId"`
	BlockID          int    `json:"blockId"`
	BlockCode        string `json:"blockCode"`
	ActivityID       int    `json:"activityId"`
	ActivityCode     string `json:"activityCode"`
	ActivityName     string `json:"activityName"`
	Category         string `json:"category"`
	SequenceNo       int    `json:"sequenceNo"`

	PlannedAreaHa    float64   `json:"plannedAreaHa"`
	PlannedStartDate time.Time `json:"-"`
	PlannedEndDate   time.Time `json:"-"`
	DurationDays     int       `json:"durationDays"`
	DailyTargetHa    float64   `json:"dailyTargetHa"`

	PlannedWorkingHours float64 `json:"plannedWorkingHours"`
	RequiredLabourDays  float64 `json:"requiredLabourDays"`
	RequiredWorkers     int     `json:"requiredWorkers"`
}

// AppliesTo reports whether an activity belongs in the programme for this kind of planting. A
// ratoon crop skips the land preparation a fresh planting needs, and the master says so.
func (a PlanActivity) AppliesTo(plantingType string) bool {
	return a.CropType == "Both" || a.CropType == plantingType
}

// GenerateTasks lays every applicable activity out over the calendar for every block.
//
// Each activity starts on the later of two dates: its standard offset from the block's planting
// date, and the day after everything it waits for has finished plus that dependency's lag. Both
// are then moved to the next working day, so nothing is scheduled on a Sunday or a holiday.
func GenerateTasks(lines []PlanLine, activities []PlanActivity, calendar WorkingCalendar) ([]PlanTask, error) {
	byID := make(map[int]PlanActivity, len(activities))
	for _, a := range activities {
		byID[a.ID] = a
	}

	ordered := append([]PlanActivity(nil), activities...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].SequenceNo != ordered[j].SequenceNo {
			return ordered[i].SequenceNo < ordered[j].SequenceNo
		}
		return ordered[i].ID < ordered[j].ID
	})

	tasks := []PlanTask{}
	for _, line := range lines {
		if line.AreaHa <= 0 {
			return nil, fmt.Errorf("block %s has no area to plan", line.BlockCode)
		}

		// Where each activity on this block finishes, so its successors can be dated. Only
		// activities that apply to this block appear, which is what lets a dependency on a skipped
		// step be ignored rather than block the chain for ever.
		endOf := map[int]time.Time{}

		for _, activity := range ordered {
			if !activity.AppliesTo(line.PlantingType) {
				continue
			}

			earliest := line.PlantingStart.AddDate(0, 0, activity.StartDayOffset)

			for _, dep := range activity.DependsOn {
				predecessor, known := byID[dep.DependsOnID]
				if !known || !predecessor.AppliesTo(line.PlantingType) {
					continue // the step it waits for is not part of this block's programme
				}
				finished, done := endOf[dep.DependsOnID]
				if !done {
					continue // out of sequence in the master; the offset alone dates this one
				}
				// The successor may start the working day after its predecessor ends, plus the lag.
				after := AddWorkingDays(NextWorkingDay(finished.AddDate(0, 0, 1), calendar),
					dep.LagDays+1, calendar)
				if after.After(earliest) {
					earliest = after
				}
			}

			start := NextWorkingDay(earliest, calendar)
			duration, err := durationDays(line.AreaHa, activity.CapacityPerDay)
			if err != nil {
				return nil, fmt.Errorf("activity %s on block %s: %w", activity.Code, line.BlockCode, err)
			}
			end := AddWorkingDays(start, duration, calendar)
			endOf[activity.ID] = end

			labourDays := Round4(line.AreaHa * activity.LabourDaysPerHa)
			workers := 0
			if labourDays > 0 {
				workers = int(math.Ceil(labourDays / float64(duration)))
			}

			tasks = append(tasks, PlanTask{
				ProjectionLineID:    line.ProjectionLineID,
				BlockID:             line.BlockID,
				BlockCode:           line.BlockCode,
				ActivityID:          activity.ID,
				ActivityCode:        activity.Code,
				ActivityName:        activity.Name,
				Category:            activity.Category,
				SequenceNo:          activity.SequenceNo,
				PlannedAreaHa:       Round4(line.AreaHa),
				PlannedStartDate:    start,
				PlannedEndDate:      end,
				DurationDays:        duration,
				DailyTargetHa:       Round4(line.AreaHa / float64(duration)),
				PlannedWorkingHours: Round4(line.AreaHa * activity.DurationPerHa),
				RequiredLabourDays:  labourDays,
				RequiredWorkers:     workers,
			})
		}
	}
	return tasks, nil
}

// durationDays is how many working days the area takes at the activity's standard capacity, never
// less than one: an activity that appears in the programme occupies at least a day of it.
func durationDays(areaHa, capacityPerDay float64) (int, error) {
	if capacityPerDay <= 0 {
		// The schema refuses an activity that needs a machine but has no capacity; this catches the
		// rest, where a zero would otherwise divide into an infinite duration.
		return 0, ErrInvalidInput
	}
	days := int(math.Ceil(areaHa / capacityPerDay))
	if days < 1 {
		days = 1
	}
	return days, nil
}

// PlanWindow is the span a generated programme covers, and what it adds up to. The header carries
// it so a list screen need not open every task to say when the work runs.
type PlanWindow struct {
	Start      time.Time
	End        time.Time
	TaskCount  int
	TotalHours float64
	TotalDays  float64
}

// Summarise reduces the tasks to the header figures.
func Summarise(tasks []PlanTask) PlanWindow {
	out := PlanWindow{TaskCount: len(tasks)}
	for i, t := range tasks {
		if i == 0 || t.PlannedStartDate.Before(out.Start) {
			out.Start = t.PlannedStartDate
		}
		if i == 0 || t.PlannedEndDate.After(out.End) {
			out.End = t.PlannedEndDate
		}
		out.TotalHours = Round4(out.TotalHours + t.PlannedWorkingHours)
		out.TotalDays = Round4(out.TotalDays + t.RequiredLabourDays)
	}
	return out
}

// ---------------------------------------------------------------- the stored plan

// The statuses a plan moves through. A Draft can be regenerated freely; releasing it hands the
// programme to the field, after which the dates are a commitment and only actuals are written.
const (
	PlanDraft    = "Draft"
	PlanReleased = "Released"
	PlanClosed   = "Closed"
)

var ValidPlanStatuses = []string{PlanDraft, PlanReleased, PlanClosed}

// ValidTaskStatuses is what a single task may be while the work is done.
var ValidTaskStatuses = []string{"Planned", "InProgress", "Completed", "Cancelled"}

// ActivityPlan is the header: which projection it came from, the calendar it was laid out against,
// and the figures the tasks add up to.
type ActivityPlan struct {
	ID                 int    `json:"id"`
	ProjectionID       int    `json:"projectionId"`
	ProjectionNo       string `json:"projectionNo"`
	ProjectionRevision int    `json:"projectionRevision"`
	ProjectionStatus   string `json:"projectionStatus"`

	PlanNo string `json:"planNo"`
	Status string `json:"status"`

	WorkOnSaturday bool     `json:"workOnSaturday"`
	WorkOnSunday   bool     `json:"workOnSunday"`
	Holidays       []string `json:"holidays"`

	StartsOn *string `json:"startsOn"`
	EndsOn   *string `json:"endsOn"`

	TaskCount         int     `json:"taskCount"`
	TotalAreaHa       float64 `json:"totalAreaHa"`
	TotalWorkingHours float64 `json:"totalWorkingHours"`
	TotalLabourDays   float64 `json:"totalLabourDays"`

	GeneratedAt *string `json:"generatedAt"`
	GeneratedBy *string `json:"generatedBy"`
	ReleasedAt  *string `json:"releasedAt"`
	ReleasedBy  *string `json:"releasedBy"`
	ClosedAt    *string `json:"closedAt"`
	ClosedBy    *string `json:"closedBy"`

	Remark    *string   `json:"remark"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy"`

	Tasks []PlanTaskRow `json:"tasks,omitempty"`

	// Derived for the screen, never stored.
	Editable bool     `json:"editable"`
	Actions  []string `json:"actions"`
}

// PlanTaskRow is one dated task as it is read back, with the block and activity resolved.
type PlanTaskRow struct {
	ID               int64  `json:"id"`
	PlanID           int    `json:"planId"`
	ProjectionLineID int    `json:"projectionLineId"`
	BlockID          int    `json:"blockId"`
	BlockCode        string `json:"blockCode"`
	BlockName        string `json:"blockName"`
	ZoneID           int    `json:"zoneId"`
	ZoneCode         string `json:"zoneCode"`
	FarmID           int    `json:"farmId"`
	FarmCode         string `json:"farmCode"`

	ActivityID   int    `json:"activityId"`
	ActivityCode string `json:"activityCode"`
	ActivityName string `json:"activityName"`
	Category     string `json:"category"`
	SequenceNo   int    `json:"sequenceNo"`

	PlannedAreaHa    float64 `json:"plannedAreaHa"`
	PlannedStartDate string  `json:"plannedStartDate"`
	PlannedEndDate   string  `json:"plannedEndDate"`
	DurationDays     int     `json:"durationDays"`
	DailyTargetHa    float64 `json:"dailyTargetHa"`

	PlannedWorkingHours float64 `json:"plannedWorkingHours"`
	RequiredLabourDays  float64 `json:"requiredLabourDays"`
	RequiredWorkers     int     `json:"requiredWorkers"`

	RequiresTractor   bool `json:"requiresTractor"`
	RequiresEquipment bool `json:"requiresEquipment"`
	RequiresMaterial  bool `json:"requiresMaterial"`
	RequiresLabour    bool `json:"requiresLabour"`

	Status            string  `json:"status"`
	CompletionPercent float64 `json:"completionPercent"`
	ActualStartDate   *string `json:"actualStartDate"`
	ActualEndDate     *string `json:"actualEndDate"`
	ActualAreaHa      float64 `json:"actualAreaHa"`
	Remark            *string `json:"remark"`
}

// PlanInput is what a caller may set: which projection to plan, and the working calendar to lay it
// out against. Every date is generated, so there is no field for one.
type PlanInput struct {
	ProjectionID   int      `json:"projectionId"`
	WorkOnSaturday *bool    `json:"workOnSaturday"`
	WorkOnSunday   *bool    `json:"workOnSunday"`
	Holidays       []string `json:"holidays"`
	Remark         *string  `json:"remark"`
	Version        int      `json:"version"`
}

// CalendarFrom turns the input into the calendar the engine takes, defaulting to the specification's
// working week when the caller says nothing.
func (in PlanInput) CalendarFrom() (WorkingCalendar, error) {
	cal := DefaultCalendar()
	if in.WorkOnSaturday != nil {
		cal.WorkOnSaturday = *in.WorkOnSaturday
	}
	if in.WorkOnSunday != nil {
		cal.WorkOnSunday = *in.WorkOnSunday
	}
	for _, raw := range in.Holidays {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", trimmed); err != nil {
			return cal, Invalid([]FieldError{{
				Field: "holidays", Message: fmt.Sprintf("%q is not a date of the form YYYY-MM-DD.", raw)}})
		}
		cal.Holidays[trimmed] = true
	}
	if !cal.WorkOnSaturday && !cal.WorkOnSunday {
		// Five-day weeks are ordinary. A calendar with no working day at all is not, and would
		// send AddWorkingDays looking for one for ever.
		return cal, nil
	}
	return cal, nil
}

// PlanActionsFor is what this caller may do to a plan in this status.
func PlanActionsFor(status string, u User) []string {
	if !u.HasAnyRole(RoleAdmin, RoleManager, RolePlanner) {
		return []string{}
	}
	switch status {
	case PlanDraft:
		if u.HasAnyRole(RoleAdmin, RoleManager) {
			return []string{"Regenerate", "Release", "Delete"}
		}
		return []string{"Regenerate"}
	case PlanReleased:
		if u.HasAnyRole(RoleAdmin, RoleManager) {
			return []string{"Close", "Reopen"}
		}
	}
	return []string{}
}
