namespace SugarcanePlanning.Domain.Enums;

/// <summary>Lifecycle of a growing season (section 4).</summary>
public enum SeasonStatus { Draft = 0, Open = 1, Closed = 2 }

/// <summary>Planting projection workflow status (section 5 / 16).</summary>
public enum ProjectionStatus
{
    Draft = 0,
    Submitted = 1,
    UnderReview = 2,
    Approved = 3,
    Rejected = 4,
    Revised = 5,
    Closed = 6
}

/// <summary>Workflow actions available on a projection (section 16).</summary>
public enum ApprovalAction
{
    Submit = 0,
    Review = 1,
    Approve = 2,
    Reject = 3,
    ReturnForCorrection = 4,
    Revise = 5,
    Close = 6
}

/// <summary>New planting versus ratoon crop (section 5).</summary>
public enum CropType { NewPlanting = 0, Ratoon = 1, Both = 2 }

/// <summary>Activity plan status (section 7).</summary>
public enum ActivityStatus
{
    NotPlanned = 0,
    Planned = 1,
    Scheduled = 2,
    InProgress = 3,
    Completed = 4,
    Delayed = 5,
    Cancelled = 6
}

/// <summary>Tractor / equipment availability (section 8, 9).</summary>
public enum AvailabilityStatus
{
    Available = 0,
    Assigned = 1,
    UnderMaintenance = 2,
    Breakdown = 3,
    Inactive = 4
}

/// <summary>Maintenance state of a machine.</summary>
public enum MaintenanceStatus
{
    Serviceable = 0,
    ServiceDue = 1,
    InService = 2,
    OutOfService = 3
}

/// <summary>Company-owned or rented asset (section 8, 9).</summary>
public enum OwnershipType { Company = 0, Rental = 1 }

/// <summary>Material categories usable in planning (section 11).</summary>
public enum MaterialCategory
{
    SeedCane = 0,
    Fertilizer = 1,
    Herbicide = 2,
    Pesticide = 3,
    Fuel = 4,
    Water = 5,
    Other = 6
}

/// <summary>Units of measure supported by the material master (section 11).</summary>
public enum UnitOfMeasure
{
    Ton = 0,
    Kilogram = 1,
    Liter = 2,
    Bag = 3,
    Piece = 4,
    Hectare = 5,
    Drum = 6
}

/// <summary>Result of a capacity comparison (section 15).</summary>
public enum CapacityStatus
{
    Sufficient = 0,
    AtRisk = 1,
    Shortage = 2,
    Unavailable = 3
}

/// <summary>Implement categories (section 9).</summary>
public enum EquipmentCategory
{
    DiscPlow = 0,
    MoldboardPlow = 1,
    Harrow = 2,
    LandLeveler = 3,
    FurrowOpener = 4,
    CanePlanter = 5,
    FertilizerSpreader = 6,
    ChemicalSprayer = 7,
    WaterTruck = 8,
    SeedCutter = 9,
    Trailer = 10,
    Other = 11
}

/// <summary>Grouping of planting activities (section 6).</summary>
public enum ActivityCategory
{
    Survey = 0,
    LandPreparation = 1,
    SeedPreparation = 2,
    Planting = 3,
    Fertilization = 4,
    WeedControl = 5,
    Irrigation = 6,
    CropCare = 7,
    Other = 8
}

/// <summary>Status of a resource booking (section 10).</summary>
public enum ScheduleStatus
{
    Planned = 0,
    Confirmed = 1,
    InProgress = 2,
    Completed = 3,
    Cancelled = 4
}

/// <summary>Soil classification used by blocks, varieties and material standards.</summary>
public enum SoilType
{
    Unspecified = 0,
    Sandy = 1,
    SandyLoam = 2,
    Loam = 3,
    ClayLoam = 4,
    Clay = 5,
    Silt = 6,
    Peat = 7,
    Volcanic = 8
}

/// <summary>Physical condition of the land (section 3).</summary>
public enum LandCondition
{
    Flat = 0,
    Undulating = 1,
    Sloped = 2,
    Rocky = 3,
    Swampy = 4
}

/// <summary>Irrigation available on a plantation block (section 3).</summary>
public enum IrrigationType
{
    None = 0,
    RainFed = 1,
    Furrow = 2,
    Sprinkler = 3,
    Drip = 4,
    CenterPivot = 5
}

/// <summary>What is currently growing on a block (section 3).</summary>
public enum CropStatus
{
    Fallow = 0,
    Prepared = 1,
    Planted = 2,
    Growing = 3,
    ReadyForHarvest = 4,
    Harvested = 5,
    Retired = 6
}

/// <summary>Skill required from a worker or operator (section 14).</summary>
public enum SkillType
{
    GeneralLabor = 0,
    TractorOperator = 1,
    EquipmentOperator = 2,
    Planter = 3,
    SprayerOperator = 4,
    Supervisor = 5,
    Driver = 6
}

/// <summary>Lifecycle of a what-if scenario (section 15).</summary>
public enum ScenarioStatus { Draft = 0, Simulated = 1, Discarded = 2, Adopted = 3 }

/// <summary>Kind of adjustment a scenario applies to the baseline plan (section 15).</summary>
public enum ScenarioAdjustmentType
{
    AdditionalRentalTractors = 0,
    AdditionalEquipment = 1,
    ExtendedWorkingHours = 2,
    ReducedPlantingArea = 3,
    ChangedPlantingDates = 4,
    ChangedActivityDuration = 5,
    AdditionalWorkers = 6
}

/// <summary>Audit-trail action verb (section 22).</summary>
public enum AuditAction
{
    Create = 0,
    Update = 1,
    Delete = 2,
    Submit = 3,
    Approve = 4,
    Reject = 5,
    Return = 6,
    Revise = 7,
    Close = 8,
    Schedule = 9,
    Reassign = 10,
    Execute = 11,
    Export = 12,
    Login = 13
}

/// <summary>Kind of conflict found by the scheduling engine (section 10).</summary>
public enum ConflictType
{
    TractorDoubleBooking = 0,
    EquipmentDoubleBooking = 1,
    OperatorDoubleBooking = 2,
    MachineUnderMaintenance = 3,
    InsufficientHorsepower = 4,
    LocationConflict = 5,
    ActivityDependency = 6,
    OutsidePlanPeriod = 7,
    IncompatibleEquipment = 8
}
