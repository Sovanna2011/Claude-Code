package domain_test

import (
	"testing"
	"time"

	"github.com/sovanna2011/sugarcane-go/backend/internal/domain"
)

func day(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

// A small programme with the shape of the real one: land preparation before planting, planting on
// the day itself, and care after — each waiting for the one before it.
func programme() []domain.PlanActivity {
	return []domain.PlanActivity{
		{ID: 1, Code: "A001", Name: "Land survey", Category: "Survey", CropType: "NewPlanting",
			SequenceNo: 1, StartDayOffset: -20, CapacityPerDay: 20, DurationPerHa: 0.5, LabourDaysPerHa: 0.2},
		{ID: 2, Code: "A002", Name: "Ploughing", Category: "LandPreparation", CropType: "NewPlanting",
			SequenceNo: 2, StartDayOffset: -10, CapacityPerDay: 5, DurationPerHa: 2, LabourDaysPerHa: 0.5,
			RequiresTractor: true,
			DependsOn:       []domain.PlanDependency{{DependsOnID: 1, IsBlocking: true}}},
		{ID: 3, Code: "A003", Name: "Planting", Category: "Planting", CropType: "Both",
			SequenceNo: 3, StartDayOffset: 0, CapacityPerDay: 4, DurationPerHa: 6, LabourDaysPerHa: 3,
			IsMandatory: true, RequiresLabour: true,
			DependsOn:   []domain.PlanDependency{{DependsOnID: 2, LagDays: 1, IsBlocking: true}}},
		{ID: 4, Code: "A004", Name: "First weeding", Category: "WeedControl", CropType: "Both",
			SequenceNo: 4, StartDayOffset: 30, CapacityPerDay: 10, DurationPerHa: 1, LabourDaysPerHa: 1,
			DependsOn: []domain.PlanDependency{{DependsOnID: 3, IsBlocking: true}}},
	}
}

func blockLine(area float64, plantingType, start string) domain.PlanLine {
	return domain.PlanLine{
		ProjectionLineID: 1, BlockID: 1, BlockCode: "BLK-001",
		AreaHa: area, PlantingType: plantingType, PlantingStart: day(start),
	}
}

func taskFor(t *testing.T, tasks []domain.PlanTask, code string) domain.PlanTask {
	t.Helper()
	for _, task := range tasks {
		if task.ActivityCode == code {
			return task
		}
	}
	t.Fatalf("no task for activity %s", code)
	return domain.PlanTask{}
}

func TestTasksAreDatedFromTheirOffsetToThePlantingDay(t *testing.T) {
	// 1 June 2026 is a Monday. Saturdays are worked, Sundays are not.
	tasks, err := domain.GenerateTasks(
		[]domain.PlanLine{blockLine(20, "NewPlanting", "2026-06-01")},
		programme(), domain.DefaultCalendar())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(tasks) != 4 {
		t.Fatalf("expected four tasks, got %d", len(tasks))
	}

	// The survey sits twenty days before planting: 12 May 2026, a Tuesday, so it needs no shifting.
	survey := taskFor(t, tasks, "A001")
	if got := survey.PlannedStartDate.Format("2006-01-02"); got != "2026-05-12" {
		t.Errorf("the survey starts %s, expected 2026-05-12", got)
	}
	// 20 ha at 20 ha a day is one working day.
	if survey.DurationDays != 1 || survey.PlannedEndDate != survey.PlannedStartDate {
		t.Errorf("the survey should take one day, got %d ending %s",
			survey.DurationDays, survey.PlannedEndDate.Format("2006-01-02"))
	}
}

func TestNothingIsScheduledOnADayThatIsNotWorked(t *testing.T) {
	calendar := domain.DefaultCalendar()
	calendar.Holidays["2026-05-12"] = true // the day the survey would otherwise start

	tasks, err := domain.GenerateTasks(
		[]domain.PlanLine{blockLine(20, "NewPlanting", "2026-06-01")},
		programme(), calendar)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if got := taskFor(t, tasks, "A001").PlannedStartDate.Format("2006-01-02"); got != "2026-05-13" {
		t.Errorf("a holiday should push the survey to the next working day, got %s", got)
	}

	// And no task anywhere may begin or end on a Sunday.
	for _, task := range tasks {
		if task.PlannedStartDate.Weekday() == time.Sunday || task.PlannedEndDate.Weekday() == time.Sunday {
			t.Errorf("%s runs %s to %s, which touches a Sunday", task.ActivityCode,
				task.PlannedStartDate.Format("2006-01-02 Mon"), task.PlannedEndDate.Format("2006-01-02 Mon"))
		}
	}
}

// The rule that makes a chain a chain: a successor cannot begin until its predecessor has finished,
// even when its own offset says it could.
func TestASuccessorWaitsForWhatItDependsOn(t *testing.T) {
	// 120 ha of ploughing at 5 ha a day is 24 working days — far longer than the ten days the
	// offsets leave between ploughing and planting, so the dependency has to push planting out.
	tasks, err := domain.GenerateTasks(
		[]domain.PlanLine{blockLine(120, "NewPlanting", "2026-06-01")},
		programme(), domain.DefaultCalendar())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	plough := taskFor(t, tasks, "A002")
	planting := taskFor(t, tasks, "A003")

	if !planting.PlannedStartDate.After(plough.PlannedEndDate) {
		t.Fatalf("planting starts %s but ploughing runs to %s",
			planting.PlannedStartDate.Format("2006-01-02"), plough.PlannedEndDate.Format("2006-01-02"))
	}
	// Its offset alone would have put it on 1 June; the chain moved it.
	if !planting.PlannedStartDate.After(day("2026-06-01")) {
		t.Errorf("the dependency did not push planting past its own offset: %s",
			planting.PlannedStartDate.Format("2006-01-02"))
	}
	// A one-day lag means one clear working day between the two.
	gap := domain.WorkingDays(plough.PlannedEndDate, planting.PlannedStartDate, domain.DefaultCalendar())
	if gap != 3 { // the end day, the lag day, and the start day
		t.Errorf("expected one clear working day between ploughing and planting, spanned %d days", gap)
	}
}

// A ratoon crop regrows from the stubble, so the land is never surveyed or ploughed for it. Those
// activities are not in its programme, and planting must not wait for steps that never happen.
func TestARatoonSkipsTheWorkItDoesNotNeed(t *testing.T) {
	tasks, err := domain.GenerateTasks(
		[]domain.PlanLine{blockLine(20, "Ratoon", "2026-06-01")},
		programme(), domain.DefaultCalendar())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("a ratoon should carry only the two activities that apply to it, got %d", len(tasks))
	}
	for _, task := range tasks {
		if task.ActivityCode == "A001" || task.ActivityCode == "A002" {
			t.Errorf("%s should not be in a ratoon's programme", task.ActivityCode)
		}
	}
	// Planting waits for ploughing, which is not in this programme — so its own offset dates it,
	// rather than the chain stalling.
	if got := taskFor(t, tasks, "A003").PlannedStartDate.Format("2006-01-02"); got != "2026-06-01" {
		t.Errorf("planting a ratoon should start on its offset, got %s", got)
	}
}

func TestDurationAndDailyTargetComeFromTheStandardCapacity(t *testing.T) {
	tasks, err := domain.GenerateTasks(
		[]domain.PlanLine{blockLine(30, "NewPlanting", "2026-06-01")},
		programme(), domain.DefaultCalendar())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Ploughing: 30 ha at 5 ha a day is six working days, and the daily target is the area spread
	// across them.
	plough := taskFor(t, tasks, "A002")
	if plough.DurationDays != 6 {
		t.Errorf("ploughing 30 ha at 5 ha a day should take 6 days, got %d", plough.DurationDays)
	}
	if plough.DailyTargetHa != 5 {
		t.Errorf("the daily target should be 5 ha, got %v", plough.DailyTargetHa)
	}
	if plough.PlannedWorkingHours != 60 {
		t.Errorf("30 ha at 2 hours a hectare is 60 hours, got %v", plough.PlannedWorkingHours)
	}

	// Planting: 3 labour days a hectare over 30 ha is 90 labour days; across 8 working days that
	// needs 12 people (90 ÷ 8 = 11.25, and a fraction of a person cannot be sent to a field).
	planting := taskFor(t, tasks, "A003")
	if planting.RequiredLabourDays != 90 {
		t.Errorf("expected 90 labour days, got %v", planting.RequiredLabourDays)
	}
	if planting.DurationDays != 8 {
		t.Fatalf("expected 8 working days of planting, got %d", planting.DurationDays)
	}
	if planting.RequiredWorkers != 12 {
		t.Errorf("expected 12 workers, got %d", planting.RequiredWorkers)
	}
}

// An area smaller than a day's capacity still occupies a day of the programme.
func TestASmallBlockStillTakesAWholeDay(t *testing.T) {
	tasks, err := domain.GenerateTasks(
		[]domain.PlanLine{blockLine(0.5, "NewPlanting", "2026-06-01")},
		programme(), domain.DefaultCalendar())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, task := range tasks {
		if task.DurationDays < 1 {
			t.Errorf("%s was scheduled for %d days", task.ActivityCode, task.DurationDays)
		}
	}
}

func TestAnActivityWithNoCapacityIsRefusedRatherThanScheduledForEver(t *testing.T) {
	broken := programme()
	broken[1].CapacityPerDay = 0
	_, err := domain.GenerateTasks(
		[]domain.PlanLine{blockLine(20, "NewPlanting", "2026-06-01")}, broken, domain.DefaultCalendar())
	if err == nil {
		t.Fatal("an activity with no daily capacity should be refused, not divided by zero")
	}
}

func TestABlockWithNoAreaIsRefused(t *testing.T) {
	_, err := domain.GenerateTasks(
		[]domain.PlanLine{blockLine(0, "NewPlanting", "2026-06-01")}, programme(), domain.DefaultCalendar())
	if err == nil {
		t.Fatal("a block with no area has nothing to plan")
	}
}

// The engine is a pure function: the same inputs must give the same programme every time, or a
// regenerated plan would differ from the one that was approved for no reason anyone could explain.
func TestGenerationIsDeterministic(t *testing.T) {
	lines := []domain.PlanLine{
		blockLine(20, "NewPlanting", "2026-06-01"),
		{ProjectionLineID: 2, BlockID: 2, BlockCode: "BLK-002", AreaHa: 45,
			PlantingType: "Ratoon", PlantingStart: day("2026-06-15")},
	}
	first, err := domain.GenerateTasks(lines, programme(), domain.DefaultCalendar())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for i := 0; i < 5; i++ {
		again, err := domain.GenerateTasks(lines, programme(), domain.DefaultCalendar())
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if len(again) != len(first) {
			t.Fatalf("run %d produced %d tasks, the first produced %d", i, len(again), len(first))
		}
		for j := range first {
			if again[j] != first[j] {
				t.Fatalf("run %d differs at task %d:\n %+v\n %+v", i, j, again[j], first[j])
			}
		}
	}
}

// Every block gets its own programme, and the two do not interfere.
func TestEachBlockIsPlannedFromItsOwnPlantingDate(t *testing.T) {
	tasks, err := domain.GenerateTasks([]domain.PlanLine{
		blockLine(20, "NewPlanting", "2026-06-01"),
		{ProjectionLineID: 2, BlockID: 2, BlockCode: "BLK-002", AreaHa: 20,
			PlantingType: "NewPlanting", PlantingStart: day("2026-07-01")},
	}, programme(), domain.DefaultCalendar())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(tasks) != 8 {
		t.Fatalf("expected four tasks for each of two blocks, got %d", len(tasks))
	}

	var first, second time.Time
	for _, task := range tasks {
		if task.ActivityCode != "A003" {
			continue
		}
		if task.BlockCode == "BLK-001" {
			first = task.PlannedStartDate
		} else {
			second = task.PlannedStartDate
		}
	}
	if !second.After(first) {
		t.Errorf("the later block should be planted later: %s then %s",
			first.Format("2006-01-02"), second.Format("2006-01-02"))
	}
}

func TestSummariseSpansEveryTask(t *testing.T) {
	tasks, err := domain.GenerateTasks(
		[]domain.PlanLine{blockLine(20, "NewPlanting", "2026-06-01")},
		programme(), domain.DefaultCalendar())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	window := domain.Summarise(tasks)

	if window.TaskCount != len(tasks) {
		t.Errorf("counted %d tasks, there are %d", window.TaskCount, len(tasks))
	}
	for _, task := range tasks {
		if task.PlannedStartDate.Before(window.Start) {
			t.Errorf("%s starts before the window", task.ActivityCode)
		}
		if task.PlannedEndDate.After(window.End) {
			t.Errorf("%s ends after the window", task.ActivityCode)
		}
	}
	if window.TotalHours <= 0 || window.TotalDays <= 0 {
		t.Errorf("the totals should add up to something: %+v", window)
	}
}
