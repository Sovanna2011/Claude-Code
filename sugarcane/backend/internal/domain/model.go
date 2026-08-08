package domain

import "time"

// Roles. Read is open to every signed-in user; writing master data and planting records is not.
const (
	RoleAdmin   = "Admin"
	RoleManager = "Manager"
	RolePlanner = "Planner"
	RoleViewer  = "Viewer"
)

// User is the caller, as carried in the bearer token.
type User struct {
	ID       int    `json:"id"`
	Username string `json:"userName"`
	FullName string `json:"fullName"`
	Role     string `json:"role"`
}

// HasAnyRole reports whether the caller holds one of the roles an endpoint asks for.
func (u User) HasAnyRole(roles ...string) bool {
	for _, r := range roles {
		if u.Role == r {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------- hierarchy

type Company struct {
	ID     int    `json:"id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type Plantation struct {
	ID          int    `json:"id"`
	CompanyID   int    `json:"companyId"`
	CompanyName string `json:"companyName"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Active      bool   `json:"active"`
}

type Farm struct {
	ID             int       `json:"id"`
	PlantationID   int       `json:"plantationId"`
	PlantationName string    `json:"plantationName"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	ManagerName    *string   `json:"managerName"`
	Remark         *string   `json:"remark"`
	Active         bool      `json:"active"`
	ZoneCount      int       `json:"zoneCount"`
	BlockCount     int       `json:"blockCount"`
	HasBoundary    bool      `json:"hasBoundary"`
	Latitude       *float64  `json:"latitude"`
	Longitude      *float64  `json:"longitude"`
	MapURL         string    `json:"mapUrl"`
	Version        int       `json:"version"`
	UpdatedAt      time.Time `json:"updatedAt"`
	UpdatedBy      string    `json:"updatedBy"`
}

type Zone struct {
	ID             int       `json:"id"`
	FarmID         int       `json:"farmId"`
	FarmName       string    `json:"farmName"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	SupervisorName *string   `json:"supervisorName"`
	Remark         *string   `json:"remark"`
	Active         bool      `json:"active"`
	BlockCount     int       `json:"blockCount"`
	HasBoundary    bool      `json:"hasBoundary"`
	Latitude       *float64  `json:"latitude"`
	Longitude      *float64  `json:"longitude"`
	MapURL         string    `json:"mapUrl"`
	Version        int       `json:"version"`
	UpdatedAt      time.Time `json:"updatedAt"`
	UpdatedBy      string    `json:"updatedBy"`
}

// NonPlantablePart is one reason a slice of a block cannot carry cane (section 5).
type NonPlantablePart struct {
	ReasonCode string  `json:"reasonCode"`
	ReasonName string  `json:"reasonName"`
	AreaHa     float64 `json:"areaHa"`
	Remark     *string `json:"remark"`
}

// Block is the lowest unit of land management and the only level that owns an area figure.
type Block struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`

	ZoneID         int    `json:"zoneId"`
	ZoneCode       string `json:"zoneCode"`
	ZoneName       string `json:"zoneName"`
	FarmID         int    `json:"farmId"`
	FarmCode       string `json:"farmCode"`
	FarmName       string `json:"farmName"`
	PlantationID   int    `json:"plantationId"`
	PlantationName string `json:"plantationName"`
	CompanyID      int    `json:"companyId"`
	CompanyName    string `json:"companyName"`

	Areas Areas `json:"areas"`

	LandStatus string `json:"landStatus"`
	CaneStatus string `json:"caneStatus"`

	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	MapURL      string   `json:"mapUrl"`
	HasBoundary bool     `json:"hasBoundary"`

	// Measured off the stored polygon rather than off the registered figure, so a survey that
	// disagrees with the paperwork is visible instead of hidden.
	BoundaryAreaHa *float64 `json:"boundaryAreaHa"`

	NonPlantableParts []NonPlantablePart `json:"nonPlantableParts"`
	Plantings         []Planting         `json:"plantings"`

	Remark    *string   `json:"remark"`
	Active    bool      `json:"active"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy"`
}

// Planting is what was planned and what happened, for one block, one crop year, one type.
type Planting struct {
	ID                  int     `json:"id"`
	BlockID             int     `json:"blockId"`
	BlockCode           string  `json:"blockCode"`
	CropSeasonID        int     `json:"cropSeasonId"`
	CropSeasonName      string  `json:"cropSeasonName"`
	CropYear            int     `json:"cropYear"`
	PlantingYear        int     `json:"plantingYear"`
	PlantingType        string  `json:"plantingType"`
	RatoonNo            int     `json:"ratoonNo"`
	CaneVarietyID       *int    `json:"caneVarietyId"`
	CaneVarietyName     *string `json:"caneVarietyName"`
	PlannedAreaHa       float64 `json:"plannedAreaHa"`
	ActualAreaHa        float64 `json:"actualAreaHa"`
	VarianceAreaHa      float64 `json:"varianceAreaHa"`
	AchievementPercent  float64 `json:"achievementPercent"`
	PlannedDate         *string `json:"plannedDate"`
	ActualDate          *string `json:"actualDate"`
	ExpectedHarvestDate *string `json:"expectedHarvestDate"`
	Remark              *string `json:"remark"`
	Version             int     `json:"version"`
}

// ---------------------------------------------------------------- reporting

// TreeNode is one row of the farm → zone → block report. The same shape carries all three levels
// so the UI5 TreeTable can bind one set of columns to the whole hierarchy.
type TreeNode struct {
	NodeType string `json:"nodeType"` // Farm | Zone | Block
	ID       int    `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`

	Areas Areas `json:"areas"`

	// Percentages of the node's own total, so a row can be read without the estate figure.
	PercentWithCane   float64 `json:"percentWithCane"`
	PercentAvailable  float64 `json:"percentAvailable"`
	PercentCannotPlan float64 `json:"percentCannotPlant"`

	LandStatus  string   `json:"landStatus,omitempty"`
	CaneStatus  string   `json:"caneStatus,omitempty"`
	VarietyName string   `json:"caneVarietyName,omitempty"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	MapURL      string   `json:"mapUrl"`

	BlockCount int         `json:"blockCount"`
	Children   []*TreeNode `json:"children"`
}

// KPI is one card on the dashboard: a hectare figure and its share of the total area.
type KPI struct {
	Key          string  `json:"key"`
	Label        string  `json:"label"`
	AreaHa       float64 `json:"areaHa"`
	PercentTotal float64 `json:"percentOfTotalArea"`
	DrillTo      string  `json:"drillTo"`
}

// ChartSeries is a named list of category/value pairs — the shape every dashboard chart binds to.
type ChartSeries struct {
	Key    string       `json:"key"`
	Title  string       `json:"title"`
	Unit   string       `json:"unit"`
	Points []ChartPoint `json:"points"`
}

type ChartPoint struct {
	Category string   `json:"category"`
	Value    float64  `json:"value"`
	Extra    *float64 `json:"extra,omitempty"` // a second measure, e.g. planned beside actual
	ID       int      `json:"id,omitempty"`    // the farm or zone the point drills into
}

// PlanActualRow compares the programme with what happened, at whatever level was asked for.
type PlanActualRow struct {
	Level              string  `json:"level"`
	ID                 int     `json:"id"`
	Code               string  `json:"code"`
	Name               string  `json:"name"`
	CropYear           int     `json:"cropYear"`
	Month              int     `json:"month,omitempty"`
	PlantingType       string  `json:"plantingType,omitempty"`
	PlannedNewHa       float64 `json:"plannedNewPlantingHa"`
	ActualNewHa        float64 `json:"actualNewPlantingHa"`
	PlannedRatoonHa    float64 `json:"plannedRatoonHa"`
	ActualRatoonHa     float64 `json:"actualRatoonHa"`
	PlannedWithCaneHa  float64 `json:"plannedAreaWithCaneHa"`
	ActualWithCaneHa   float64 `json:"actualAreaWithCaneHa"`
	PlannedAvailableHa float64 `json:"plannedAvailableAreaHa"`
	ActualAvailableHa  float64 `json:"actualAvailableAreaHa"`
	VarianceHa         float64 `json:"varianceAreaHa"`
	AchievementPercent float64 `json:"achievementPercent"`
}

// MapFeature is one GeoJSON feature for the dashboard map, with the block's figures attached so a
// click can fill the detail panel without a second request.
type MapFeature struct {
	Type       string         `json:"type"`
	Geometry   any            `json:"geometry"`
	Properties map[string]any `json:"properties"`
}

type MapFeatureCollection struct {
	Type     string       `json:"type"`
	Features []MapFeature `json:"features"`
}

// ---------------------------------------------------------------- planning master data

// Season is a growing season: the year, and the windows planting and harvesting must fall inside.
// A window may wrap the year end — October to February is an ordinary season south of the equator.
type Season struct {
	ID                  int     `json:"id"`
	Code                string  `json:"code"`
	Name                string  `json:"name"`
	CropYear            int     `json:"cropYear"`
	StartsOn            string  `json:"startsOn"`
	EndsOn              string  `json:"endsOn"`
	PlantingWindowStart *string `json:"plantingWindowStart"`
	PlantingWindowEnd   *string `json:"plantingWindowEnd"`
	HarvestWindowStart  *string `json:"harvestWindowStart"`
	HarvestWindowEnd    *string `json:"harvestWindowEnd"`
	Status              string  `json:"status"`
	Remark              *string `json:"remark"`
	Active              bool    `json:"active"`
	Version             int     `json:"version"`
}

type SeasonInput struct {
	Code                string  `json:"code"`
	Name                string  `json:"name"`
	CropYear            int     `json:"cropYear"`
	StartsOn            string  `json:"startsOn"`
	EndsOn              string  `json:"endsOn"`
	PlantingWindowStart *string `json:"plantingWindowStart"`
	PlantingWindowEnd   *string `json:"plantingWindowEnd"`
	HarvestWindowStart  *string `json:"harvestWindowStart"`
	HarvestWindowEnd    *string `json:"harvestWindowEnd"`
	Status              string  `json:"status"`
	Remark              *string `json:"remark"`
	Active              *bool   `json:"active"`
	Version             int     `json:"version"`
}

// Variety carries the three figures the projection formulas read from it: the seed rate that sizes
// the seed-cane requirement, and the yield and loss that turn an area into expected tonnes.
type Variety struct {
	ID                  int     `json:"id"`
	Code                string  `json:"code"`
	Name                string  `json:"name"`
	GrowingPeriodMonths int     `json:"growingPeriodMonths"`
	SeedRatePerHa       float64 `json:"seedRatePerHa"`
	ExpectedYieldPerHa  float64 `json:"expectedYieldPerHa"`
	ExpectedLossPercent float64 `json:"expectedLossPercent"`
	Remark              *string `json:"remark"`
	Active              bool    `json:"active"`
	Version             int     `json:"version"`
}

type VarietyInput struct {
	Code                string  `json:"code"`
	Name                string  `json:"name"`
	GrowingPeriodMonths int     `json:"growingPeriodMonths"`
	SeedRatePerHa       float64 `json:"seedRatePerHa"`
	ExpectedYieldPerHa  float64 `json:"expectedYieldPerHa"`
	ExpectedLossPercent float64 `json:"expectedLossPercent"`
	Remark              *string `json:"remark"`
	Active              *bool   `json:"active"`
	Version             int     `json:"version"`
}

// Activity is one step of the planting programme, with everything the engines need to turn an area
// into a duration, a machine count and a workforce.
type Activity struct {
	ID                 int    `json:"id"`
	CompanyID          int    `json:"companyId"`
	Code               string `json:"code"`
	Name               string `json:"name"`
	Category           string `json:"category"`
	ApplicableCropType string `json:"applicableCropType"`
	SequenceNo         int    `json:"sequenceNo"`

	StandardStartDayOffset  int     `json:"standardStartDayOffset"`
	StandardCapacityPerHour float64 `json:"standardCapacityPerHour"`
	StandardCapacityPerDay  float64 `json:"standardCapacityPerDay"`
	StandardDurationPerHa   float64 `json:"standardDurationPerHa"`
	StandardLabourDaysPerHa float64 `json:"standardLabourDaysPerHa"`

	RequiredTractorType       *string `json:"requiredTractorType"`
	RequiredEquipmentCategory *string `json:"requiredEquipmentCategory"`

	IsMandatory       bool `json:"isMandatory"`
	RequiresTractor   bool `json:"requiresTractor"`
	RequiresEquipment bool `json:"requiresEquipment"`
	RequiresMaterial  bool `json:"requiresMaterial"`
	RequiresLabour    bool `json:"requiresLabour"`
	AllowOverlap      bool `json:"allowOverlap"`

	Dependencies []Dependency `json:"dependencies"`

	Remark    *string   `json:"remark"`
	Active    bool      `json:"active"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy"`
}

type ActivityInput struct {
	CompanyID                 int     `json:"companyId"`
	Code                      string  `json:"code"`
	Name                      string  `json:"name"`
	Category                  string  `json:"category"`
	ApplicableCropType        string  `json:"applicableCropType"`
	SequenceNo                int     `json:"sequenceNo"`
	StandardStartDayOffset    int     `json:"standardStartDayOffset"`
	StandardCapacityPerHour   float64 `json:"standardCapacityPerHour"`
	StandardCapacityPerDay    float64 `json:"standardCapacityPerDay"`
	StandardDurationPerHa     float64 `json:"standardDurationPerHa"`
	StandardLabourDaysPerHa   float64 `json:"standardLabourDaysPerHa"`
	RequiredTractorType       *string `json:"requiredTractorType"`
	RequiredEquipmentCategory *string `json:"requiredEquipmentCategory"`
	IsMandatory               bool    `json:"isMandatory"`
	RequiresTractor           bool    `json:"requiresTractor"`
	RequiresEquipment         bool    `json:"requiresEquipment"`
	RequiresMaterial          bool    `json:"requiresMaterial"`
	RequiresLabour            bool    `json:"requiresLabour"`
	AllowOverlap              bool    `json:"allowOverlap"`
	Remark                    *string `json:"remark"`
	Active                    *bool   `json:"active"`
	Version                   int     `json:"version"`
}

// Dependency is "this activity waits for that one", with an optional lag. A blocking dependency
// stops the successor; an advisory one only warns.
type Dependency struct {
	ID                  int     `json:"id"`
	ActivityID          int     `json:"activityId"`
	DependsOnID         int     `json:"dependsOnId"`
	DependsOnCode       string  `json:"dependsOnCode"`
	DependsOnName       string  `json:"dependsOnName"`
	DependsOnSequenceNo int     `json:"dependsOnSequenceNo"`
	LagDays             int     `json:"lagDays"`
	IsBlocking          bool    `json:"isBlocking"`
	Remark              *string `json:"remark"`
}

type DependencyInput struct {
	ActivityID  int     `json:"activityId"`
	DependsOnID int     `json:"dependsOnId"`
	LagDays     int     `json:"lagDays"`
	IsBlocking  bool    `json:"isBlocking"`
	Remark      *string `json:"remark"`
}

// The vocabularies the activity master uses, checked before a value reaches an enum column.
var (
	ValidActivityCategories = []string{"Survey", "LandPreparation", "SeedPreparation", "Planting",
		"Fertilization", "WeedControl", "PestControl", "Irrigation", "CropCare", "Harvesting", "Other"}
	ValidApplicableCropTypes = []string{"NewPlanting", "Ratoon", "Both"}
	ValidSeasonStatuses      = []string{"Planned", "Open", "Closed"}
)
