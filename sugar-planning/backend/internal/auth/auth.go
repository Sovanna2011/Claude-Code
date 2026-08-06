// Package auth holds the authorisation model: who the caller is, what they may
// do and which companies and factories they may see.
//
// Authentication is delegated to the enterprise identity provider (see
// oidc.go). This file is about what happens after a token has been verified.
package auth

import (
	"context"
	"fmt"
	"sort"

	"github.com/kss/sugarplan/internal/domain"
)

// Role codes. These are the roles listed in section 19 of the specification.
const (
	RoleSystemAdmin        = "SYSTEM_ADMIN"
	RoleMasterDataAdmin    = "MASTER_DATA_ADMIN"
	RoleProductionPlanner  = "PRODUCTION_PLANNER"
	RoleCaneOperator       = "CANE_OPERATOR"
	RoleShiftSupervisor    = "SHIFT_SUPERVISOR"
	RoleProductionOperator = "PRODUCTION_OPERATOR"
	RoleWarehouseOperator  = "WAREHOUSE_OPERATOR"
	RoleQualityUser        = "QUALITY_USER"
	RoleMaintenanceUser    = "MAINTENANCE_USER"
	RoleShipmentPlanner    = "SHIPMENT_PLANNER"
	RoleCostController     = "COST_CONTROLLER"
	RoleApprover           = "APPROVER"
	RoleExecutiveViewer    = "EXECUTIVE_VIEWER"
	RoleAuditor            = "AUDITOR"
)

// DefaultRoles is the role to permission mapping. It is the executable form of
// the authorisation matrix in docs/03-module-and-authorization-matrix.md.
//
// Two principles are visible in the table. Least privilege: an operator can
// post the actuals for their own area and nothing else. Separation of duties:
// the planner who builds and submits a plan cannot approve or release it, and
// the approver cannot edit the plan they are approving.
var DefaultRoles = map[string][]string{
	RoleSystemAdmin: {
		domain.PermAdmin, domain.PermMasterDataRead, domain.PermMasterDataWrite,
		domain.PermPlanRead, domain.PermReportRead, domain.PermAuditRead,
	},
	RoleMasterDataAdmin: {
		domain.PermMasterDataRead, domain.PermMasterDataWrite, domain.PermPlanRead,
		domain.PermReportRead,
	},
	RoleProductionPlanner: {
		domain.PermMasterDataRead, domain.PermPlanRead, domain.PermPlanWrite,
		domain.PermPlanSubmit, domain.PermMaterialsRead, domain.PermReportRead,
	},
	RoleCaneOperator: {
		domain.PermMasterDataRead, domain.PermPlanRead, domain.PermActualCane, domain.PermReportRead,
	},
	RoleShiftSupervisor: {
		domain.PermMasterDataRead, domain.PermPlanRead, domain.PermActualCane,
		domain.PermActualProduce, domain.PermDowntimeWrite, domain.PermReportRead,
	},
	RoleProductionOperator: {
		domain.PermMasterDataRead, domain.PermPlanRead, domain.PermActualProduce, domain.PermReportRead,
	},
	RoleWarehouseOperator: {
		domain.PermMasterDataRead, domain.PermPlanRead, domain.PermActualStock,
		domain.PermActualShip, domain.PermReportRead,
	},
	RoleQualityUser: {
		domain.PermMasterDataRead, domain.PermPlanRead, domain.PermQualityWrite,
		domain.PermQualityRelease, domain.PermReportRead,
	},
	RoleMaintenanceUser: {
		domain.PermMasterDataRead, domain.PermPlanRead, domain.PermDowntimeWrite, domain.PermReportRead,
	},
	RoleShipmentPlanner: {
		domain.PermMasterDataRead, domain.PermPlanRead, domain.PermPlanWrite,
		domain.PermActualShip, domain.PermReportRead,
	},
	RoleCostController: {
		domain.PermMasterDataRead, domain.PermPlanRead, domain.PermReportRead,
	},
	RoleApprover: {
		domain.PermMasterDataRead, domain.PermPlanRead, domain.PermPlanApprove,
		domain.PermPlanRelease, domain.PermPlanReopen, domain.PermCapacityOverride,
		domain.PermReportRead,
	},
	RoleExecutiveViewer: {
		domain.PermPlanRead, domain.PermMasterDataRead, domain.PermReportRead,
	},
	RoleAuditor: {
		domain.PermPlanRead, domain.PermMasterDataRead, domain.PermReportRead, domain.PermAuditRead,
	},
}

// Principal is the authenticated caller.
type Principal struct {
	Subject     string   `json:"subject"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Email       string   `json:"email,omitempty"`
	Roles       []string `json:"roles"`
	// Companies and Factories are the data scope. An empty Factories list with
	// a non-empty Companies list means "every factory of those companies".
	Companies []string `json:"companies"`
	Factories []string `json:"factories"`

	permissions map[string]bool
}

// NewPrincipal resolves the caller's roles into permissions.
func NewPrincipal(subject, username, displayName, email string, roles, companies, factories []string) Principal {
	p := Principal{
		Subject: subject, Username: username, DisplayName: displayName, Email: email,
		Roles: roles, Companies: companies, Factories: factories,
		permissions: map[string]bool{},
	}
	for _, r := range roles {
		for _, perm := range DefaultRoles[r] {
			p.permissions[perm] = true
		}
	}
	return p
}

// Can reports whether the caller holds a permission. A system administrator
// holds every permission except the ones that would breach separation of
// duties, which are granted explicitly rather than inherited.
func (p Principal) Can(permission string) bool {
	if p.permissions == nil {
		return false
	}
	return p.permissions[permission]
}

// Permissions lists the caller's permissions, sorted, for the session profile
// endpoint that the SAPUI5 shell uses to enable and disable actions.
func (p Principal) Permissions() []string {
	out := make([]string, 0, len(p.permissions))
	for perm := range p.permissions {
		out = append(out, perm)
	}
	sort.Strings(out)
	return out
}

// Require returns a domain error when the permission is missing, so callers can
// simply return it.
func (p Principal) Require(permission string) error {
	if p.Can(permission) {
		return nil
	}
	return fmt.Errorf("%w: %s requires the %s permission", domain.ErrForbidden, p.Username, permission)
}

// CanSeeFactory applies the data scope. It fails closed: a caller with no
// scope at all sees nothing.
func (p Principal) CanSeeFactory(factoryID string) bool {
	if factoryID == "" {
		return true // the request is not factory specific
	}
	for _, f := range p.Factories {
		if f == factoryID {
			return true
		}
	}
	return false
}

// CanSeeCompany applies the company-level data scope.
func (p Principal) CanSeeCompany(companyID string) bool {
	if companyID == "" {
		return true
	}
	for _, c := range p.Companies {
		if c == companyID {
			return true
		}
	}
	return false
}

// RequireFactory returns a domain error when the factory is out of scope.
func (p Principal) RequireFactory(factoryID string) error {
	if p.CanSeeFactory(factoryID) {
		return nil
	}
	return fmt.Errorf("%w: %s has no access to factory %s", domain.ErrForbidden, p.Username, factoryID)
}

// IsAnonymous reports whether the principal is unauthenticated.
func (p Principal) IsAnonymous() bool { return p.Subject == "" }

// ---------------------------------------------------------------------------
// Context plumbing
// ---------------------------------------------------------------------------

type principalKey struct{}

// WithPrincipal stores the caller in the request context.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// FromContext retrieves the caller. The zero principal is anonymous and holds
// no permissions, so a handler that forgets the middleware fails closed.
func FromContext(ctx context.Context) Principal {
	if p, ok := ctx.Value(principalKey{}).(Principal); ok {
		return p
	}
	return Principal{}
}

// KnownRoles lists the configured role codes, sorted.
func KnownRoles() []string {
	out := make([]string, 0, len(DefaultRoles))
	for r := range DefaultRoles {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}
