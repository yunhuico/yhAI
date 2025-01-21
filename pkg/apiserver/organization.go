package apiserver

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/permission"
)

// ListJoinedOrgs list the orgs that current user has joined
// @Summary list the orgs that current user has joined
// @Produce json
// @Success 200 {object} apiserver.R{data=[]model.Organization}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/joined [get]
func (h *APIHandler) ListJoinedOrgs(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	userID := getSession(c).UserID
	organizations, err := h.db.GetUserJoinedOrganizations(ctx, userID)
	if err != nil {
		err = fmt.Errorf("quering organizations: %w", err)
		return
	}

	OK(c, organizations)
}

// ListJoinedOrgsAndRoles list the orgs that current user has joined and the user's role in the orgs
// @Summary list the orgs that current user has joined and the user's role in the orgs
// @Produce json
// @Success 200 {object} apiserver.R{data=[]response.GetOrgByIDResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/joined/role [get]
func (h *APIHandler) ListJoinedOrgsAndRoles(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	userID := getSession(c).UserID
	memberWithOrgs, err := h.db.GetOrgsJoinedByUser(ctx, userID)
	if err != nil {
		err = fmt.Errorf("quering organizations: %w", err)
		return
	}

	orgWithRoles := make([]response.GetOrgByIDResp, len(memberWithOrgs))
	var role permission.Role

	for i := range memberWithOrgs {
		role, err = permission.FindRole(memberWithOrgs[i].Role)
		if err != nil {
			err = fmt.Errorf("findingRole %q: %w", memberWithOrgs[i].Role, err)
			return
		}
		orgWithRoles[i] = response.GetOrgByIDResp{
			Organization: memberWithOrgs[i].Organization,
			Role:         string(role.Name()),
			Permissions:  role.Permissions(),
		}
	}

	OK(c, orgWithRoles)
}

// GetOrgByID gets org detail by its id, where the org must be visible to the user
// @Summary gets org detail by its id, where the org must be visible to the user
// @Produce json
// @Param   id  path string true "organization id"
// @Success 200 {object} apiserver.R{data=response.GetOrgByIDResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/{id} [get]
func (h *APIHandler) GetOrgByID(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	orgID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("binding org id: %w", err)
		return
	}
	userID := getSession(c).UserID

	role, err := h.enforcer.EnsurePermissionsReturningRole(ctx, userID, model.OwnerRef{
		OwnerType: model.OwnerTypeOrganization,
		OwnerID:   orgID,
	}, permission.OrgRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	organization, err := h.db.GetOrganizationByID(ctx, orgID)
	if err != nil {
		err = fmt.Errorf("querying organizaiton: %w", err)
		return
	}

	OK(c, response.GetOrgByIDResp{
		Organization: organization,
		Role:         string(role.Name()),
		Permissions:  role.Permissions(),
	})
}

// CreateOrg user creates new org
// @Summary user creates new org
// @Produce json
// @Param   body body model.EditableOrganization true "HTTP body"
// @Success 200 {object} apiserver.R{data=model.Organization}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs [post]
func (h *APIHandler) CreateOrg(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req model.EditableOrganization
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}
	userID := getSession(c).UserID

	org := model.Organization{
		EditableOrganization: req,
	}
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.InsertOrganization(ctx, &org)
		if err != nil {
			err = fmt.Errorf("inserting organization: %w", err)
			return
		}
		err = tx.InsertOrgMember(ctx, &model.OrganizationMember{
			OrgID:  org.ID,
			UserID: userID,
			Role:   model.OrgMemberRoleOwner,
		})
		if err != nil {
			err = fmt.Errorf("granting user owner of the new org: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("transaction aborted: %w", err)
		return
	}

	OK(c, org)
}

// UpdateOrgByID update org attributes
// @Summary update org attributes
// @Produce json
// @Param id path int true "Organization ID"
// @Param   body body model.EditableOrganization true "HTTP body"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/{id} [put]
func (h *APIHandler) UpdateOrgByID(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req model.EditableOrganization
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	orgID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("binding org id: %w", err)
		return
	}
	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}
	userID := getSession(c).UserID

	owner := model.OwnerRef{
		OwnerType: model.OwnerTypeOrganization,
		OwnerID:   orgID,
	}

	err = h.enforcer.EnsurePermissions(ctx, userID, owner, permission.OrgUpdate)
	if err != nil {
		err = fmt.Errorf("ensuring permission: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.UpdateOrganizationByID(ctx, &model.Organization{
			ID:                   orgID,
			EditableOrganization: req,
		})
		if err != nil {
			err = fmt.Errorf("updating organization: %w", err)
			return
		}
		err = tx.UpdateTemplateAuthorInfoByOwner(ctx, owner, req.Name, req.AvatarURL)
		if err != nil {
			err = fmt.Errorf("updating template author info: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("transaction aborted: %w", err)
		return
	}
	err = h.db.UpdateOrganizationByID(ctx, &model.Organization{
		ID:                   orgID,
		EditableOrganization: req,
	})
	if err != nil {
		err = fmt.Errorf("updating organization: %w", err)
		return
	}

	OK(c, nil)
}

// ListOrgMembers list org members by org id, where the org must be visible to the user
// @Summary list org members by org id, where the org must be visible to the user
// @Produce json
// @Param id path int true "Organization ID"
// @Success 200 {object} apiserver.R{data=response.ListOrgMembersResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/{id}/members [get]
func (h *APIHandler) ListOrgMembers(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	limit, offset, _, err := extractPageParameters(c)
	if err != nil {
		err = fmt.Errorf("binding paging parms: %w", err)
		return
	}

	orgID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("binding org id: %w", err)
		return
	}
	userID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, userID, model.OwnerRef{
		OwnerType: model.OwnerTypeOrganization,
		OwnerID:   orgID,
	}, permission.MemberRead)
	if err != nil {
		err = fmt.Errorf("ensuring permission: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	users, count, err := h.db.ListOrgUsers(ctx, orgID, limit, offset)
	if err != nil {
		err = fmt.Errorf("listing users: %w", err)
		return
	}

	OK(c, response.ListOrgMembersResp{
		Total: count,
		Users: users,
	})
}

// ChangeOrgMemberRole change a specified member's role in the org
//
// A member can not change their own role.
// An owner can make the other member the organization owner,
// resulting the original owner steps down as Maintainer.
// @Summary change a specified member's role in the org
// @Description A member can not change their own role. An owner can make the other member the organization owner, resulting the original owner steps down as Maintainer.
// @Produce json
// @Param id path int true "Organization ID"
// @Param userId path int true "Member user ID"
// @Param body body payload.ChangeOrgMemberRoleReq true "HTTP body"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/{id}/members/{userId}/role [put]
func (h *APIHandler) ChangeOrgMemberRole(c *gin.Context) {
	var (
		req payload.ChangeOrgMemberRoleReq
		err error
		ctx = c.Request.Context()
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	orgID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("binding org id: %w", err)
		return
	}
	memberUserID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		err = fmt.Errorf("binding userId: %w", err)
		return
	}
	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}
	newRole, err := permission.FindRole(req.NewRole)
	if err != nil {
		err = fmt.Errorf("binding newRole: %w", err)
		return
	}

	operatorUserID := getSession(c).UserID
	operatorRole, err := h.enforcer.EnsurePermissionsReturningRole(ctx, operatorUserID, model.OwnerRef{
		OwnerType: model.OwnerTypeOrganization,
		OwnerID:   orgID,
	}, permission.MemberChangeRole)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}
	if !operatorRole.GreaterOrEqual(newRole) {
		err = fmt.Errorf("not enough permission level: %s cannot grant %s", operatorRole.Name(), newRole.Name())
		return
	}

	// is the user a member of the organization?
	memberUser, err := h.db.GetOrgMemberByID(ctx, orgID, memberUserID)
	if errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("user %d is not a member of org %d", memberUserID, orgID)
		return
	}
	if err != nil {
		err = fmt.Errorf("inspecting user to be modified: %w", err)
		return
	}

	// an owner is always an owner unless they want to step down themselves.
	if memberUser.Role == model.OrgMemberRoleOwner {
		err = fmt.Errorf("user %d is the owner of org %d", memberUser.UserID, memberUser.OrgID)
		return
	}

	// usual scenarios
	if !newRole.Equals(permission.Owner) {
		err = h.db.UpdateOrgMemberRoleByID(ctx, orgID, memberUserID, newRole.Name())
		if err != nil {
			err = fmt.Errorf("updating member's role: %w", err)
			return
		}

		OK(c, nil)

		return
	}

	// the owner is stepping down
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.UpdateOrgMemberRoleByID(ctx, orgID, operatorUserID, model.OrgMemberRoleMaintainer)
		if err != nil {
			err = fmt.Errorf("stepping down the original owner: %w", err)
			return
		}
		err = tx.UpdateOrgMemberRoleByID(ctx, orgID, memberUserID, model.OrgMemberRoleOwner)
		if err != nil {
			err = fmt.Errorf("updating member's role to owner: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("transaction aborted: %w", err)
		return
	}

	OK(c, nil)
}

// RemoveMembersFromOrg remove members from the org.
// An owner can not be deleted.
// A user can not delete themselves.
// @Summary remove members from the org.
// @Description An owner can not be deleted. A user can not delete themselves.
// @Produce json
// @Param id path int true "Organization ID"
// @Param body body payload.RemoveMembersFromOrgReq true "HTTP body"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/{id}/members [delete]
func (h *APIHandler) RemoveMembersFromOrg(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req payload.RemoveMembersFromOrgReq
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	orgID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("binding org id: %w", err)
		return
	}
	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}
	userID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, userID, model.OwnerRef{
		OwnerType: model.OwnerTypeOrganization,
		OwnerID:   orgID,
	}, permission.MemberDelete)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	for _, item := range req.UserIDs {
		if item == userID {
			err = errors.New("you cannot delete yourself from the org. Use leave org instead")
			return
		}
	}

	members, err := h.db.GetOrgMembersByUserIDs(ctx, orgID, req.UserIDs...)
	if err != nil {
		err = fmt.Errorf("querying members: %w", err)
		return
	}
	if len(members) == 0 {
		OK(c, nil)
		return
	}
	for _, item := range members {
		if item.Role == model.OrgMemberRoleOwner {
			err = fmt.Errorf("user %d is the organiztion owner and can not be deleted", item.UserID)
			return
		}
	}

	err = h.db.DeleteOrgMemberByUserIDs(ctx, orgID, req.UserIDs...)
	if err != nil {
		err = fmt.Errorf("deleting members: %w", err)
		return
	}

	OK(c, nil)
}

// DeleteOrgByID delete an org.
// Effectively, Only an owner can delete their org.
// @Summary delete an org. Effectively, Only an owner can delete their org.
// @Produce json
// @Param id path int true "Organization ID"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/{id} [delete]
func (h *APIHandler) DeleteOrgByID(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	orgID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("binding org id: %w", err)
		return
	}
	userID := getSession(c).UserID

	ownerRef := model.OwnerRef{
		OwnerType: model.OwnerTypeOrganization,
		OwnerID:   orgID,
	}
	err = h.enforcer.EnsurePermissions(ctx, userID, ownerRef, permission.OrgDelete)
	if err != nil {
		err = fmt.Errorf("ensuring permission: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	// Wow! lots of tables are touched.
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = h.db.DeleteConfirmsByOwner(ctx, ownerRef)
		if err != nil {
			err = fmt.Errorf("deleting confirms: %w", err)
			return
		}
		err = h.db.DeleteCredentialsByOwner(ctx, ownerRef)
		if err != nil {
			err = fmt.Errorf("deleting credentials: %w", err)
			return
		}
		err = h.db.DeleteKVsByOwner(ctx, ownerRef)
		if err != nil {
			err = fmt.Errorf("deleting KVs: %w", err)
			return
		}
		err = h.db.DeleteNodesByOwner(ctx, ownerRef)
		if err != nil {
			err = fmt.Errorf("deleting nodes: %w", err)
			return
		}
		err = h.db.DeleteTriggersByOwner(ctx, ownerRef)
		if err != nil {
			err = fmt.Errorf("deleting triggers: %w", err)
			return
		}
		err = h.db.DeleteWorkflowInstancesByOwner(ctx, ownerRef)
		if err != nil {
			err = fmt.Errorf("deleting workflow instances: %w", err)
			return
		}
		// To ensure other db.DeleteXXXByOwner methods work,
		// workflows must be deleted finally, before deleting organizations.
		err = h.db.DeleteWorkflowsByOwner(ctx, ownerRef)
		if err != nil {
			err = fmt.Errorf("deleting workflows: %w", err)
			return
		}
		err = h.db.DeleteOrgInviteLinksByOrg(ctx, orgID)
		if err != nil {
			err = fmt.Errorf("deleting org invite links: %w", err)
			return
		}
		err = h.db.DeleteOrgMembersByOrgID(ctx, orgID)
		if err != nil {
			err = fmt.Errorf("deleting org members: %w", err)
			return
		}
		err = h.db.DeleteOrganizationByID(ctx, orgID)
		if err != nil {
			err = fmt.Errorf("deleting org: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("transaction aborted: %w", err)
		return
	}

	OK(c, nil)
}

// GetOrgInviteLinks get invite links of the org for all roles
// @Summary get invite links of the org for all roles
// @Produce json
// @Param id path int true "Organization ID"
// @Success 200 {object} apiserver.R{data=[]model.OrganizationInviteLink}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/{id}/inviteLinks [get]
func (h *APIHandler) GetOrgInviteLinks(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	orgID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("binding org id: %w", err)
		return
	}
	userID := getSession(c).UserID

	ownerRef := model.OwnerRef{
		OwnerType: model.OwnerTypeOrganization,
		OwnerID:   orgID,
	}
	err = h.enforcer.EnsurePermissions(ctx, userID, ownerRef, permission.InviteLinkRead)
	if err != nil {
		err = fmt.Errorf("ensuring permission: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	links, err := h.db.GetOrgInviteLinksByOrg(ctx, orgID)
	if err != nil {
		err = fmt.Errorf("quering invite links: %w", err)
		return
	}

	OK(c, links)
}

// ResetOrgInviteLinks reset invite links of the org for all roles
// @Summary reset invite links of the org for all roles
// @Produce json
// @Param id path int true "Organization ID"
// @Success 200 {object} apiserver.R{data=[]model.OrganizationInviteLink}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/{id}/inviteLinks [put]
func (h *APIHandler) ResetOrgInviteLinks(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	orgID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("binding org id: %w", err)
		return
	}
	userID := getSession(c).UserID

	ownerRef := model.OwnerRef{
		OwnerType: model.OwnerTypeOrganization,
		OwnerID:   orgID,
	}
	err = h.enforcer.EnsurePermissions(ctx, userID, ownerRef, permission.InviteLinkWrite, permission.InviteLinkRead)
	if err != nil {
		err = fmt.Errorf("ensuring permission: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	var links []model.OrganizationInviteLink
	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.DeleteOrgInviteLinksByOrg(ctx, orgID)
		if err != nil {
			err = fmt.Errorf("deleting invite links: %w", err)
			return
		}
		links, err = tx.InsertOrgInviteLinksByOrg(ctx, orgID)
		if err != nil {
			err = fmt.Errorf("generating invite links: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("transaction aborted: %w", err)
		return
	}

	OK(c, links)
}

// DeleteOrgInviteLinks delete all invite links of the org
// @Summary delete all invite links of the org
// @Produce json
// @Param id path int true "Organization ID"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/{id}/inviteLinks [delete]
func (h *APIHandler) DeleteOrgInviteLinks(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	orgID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("binding org id: %w", err)
		return
	}
	userID := getSession(c).UserID

	ownerRef := model.OwnerRef{
		OwnerType: model.OwnerTypeOrganization,
		OwnerID:   orgID,
	}
	err = h.enforcer.EnsurePermissions(ctx, userID, ownerRef, permission.InviteLinkWrite)
	if err != nil {
		err = fmt.Errorf("ensuring permission: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	err = h.db.DeleteOrgInviteLinksByOrg(ctx, orgID)
	if err != nil {
		err = fmt.Errorf("deleting invite links: %w", err)
		return
	}

	OK(c, nil)
}

// LeaveOrg a member leaves org by themselves.
// For owner, a successor must be assigned.
// @Summary a member leaves org by themselves. For owner, a successor must be assigned.
// @Produce json
// @Param id path int true "Organization ID"
// @Param body body payload.LeaveOrgReq true "Successor info, successorUserId can be 0 if the user is not an owner."
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/{id}/members/me [delete]
func (h *APIHandler) LeaveOrg(c *gin.Context) {
	var (
		req payload.LeaveOrgReq
		err error
		ctx = c.Request.Context()
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	orgID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("binding org id: %w", err)
		return
	}
	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}
	userID := getSession(c).UserID

	role, err := h.enforcer.EnsurePermissionsReturningRole(ctx, userID, model.OwnerRef{
		OwnerType: model.OwnerTypeOrganization,
		OwnerID:   orgID,
	}, permission.OrgRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}
	if !role.Equals(permission.Owner) {
		err = h.db.DeleteOrgMemberByUserIDs(ctx, orgID, userID)
		if err != nil {
			err = fmt.Errorf("deleting user from org: %w", err)
			return
		}

		OK(c, nil)
		return
	}

	// organization owner is leaving
	if req.SuccessorUserID == 0 {
		err = errors.New("organization owner successor user id must be specified")
		return
	}
	if req.SuccessorUserID == userID {
		err = errors.New("you need to find someone other than yourself to be the successor")
		return
	}
	successor, err := h.db.GetOrgMemberByID(ctx, orgID, req.SuccessorUserID)
	if errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("user %d is not a member of the org", req.SuccessorUserID)
		return
	}
	if err != nil {
		err = fmt.Errorf("querying successor: %w", err)
		return
	}

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.DeleteOrgMemberByUserIDs(ctx, orgID, userID)
		if err != nil {
			err = fmt.Errorf("deleting original owner: %w", err)
			return
		}
		err = tx.UpdateOrgMemberRoleByID(ctx, orgID, successor.UserID, model.OrgMemberRoleOwner)
		if err != nil {
			err = fmt.Errorf("granting the sucessor owner: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("transaction aborted: %w", err)
		return
	}

	OK(c, nil)
}

// AcceptOrgInvite a user accept org invitation.
// A user can not join the same org twice.
// @Summary a user accept org invitation.
// @Description A user can not join the same org twice.
// @Produce json
// @Param id path string true "Invite link id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/inviteLinks/{id}/accept [post]
func (h *APIHandler) AcceptOrgInvite(c *gin.Context) {
	var (
		err    error
		ctx    = c.Request.Context()
		linkID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if linkID == "" {
		err = errBizInvitationNotFound
		return
	}
	link, err := h.db.GetOrgInviteLinkByID(ctx, linkID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errBizInvitationNotFound
		return
	}
	if err != nil {
		err = fmt.Errorf("querying invite link: %w", err)
		return
	}

	// is the organization still present?
	org, err := h.db.GetOrganizationByID(ctx, link.OrgID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errBizInvitationNotFound
		return
	}
	if err != nil {
		err = fmt.Errorf("querying related org: %w", err)
		return
	}

	userID := getSession(c).UserID
	_, err = h.db.GetOrgMemberByID(ctx, link.OrgID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		// relax
	} else if err == nil {
		var bizErr = errBizUserAlreadyOrgMember
		c.PureJSON(bizErr.StatusCode(), R{
			Code: bizErr.Code(),
			Msg:  bizErr.Error(),
			Data: org,
		})

		return
	} else {
		err = fmt.Errorf("querying user org membership: %w", err)
		return
	}

	err = h.db.InsertOrgMember(ctx, &model.OrganizationMember{
		OrgID:  link.OrgID,
		UserID: userID,
		Role:   link.Role,
	})
	if err != nil {
		err = fmt.Errorf("granting user access: %w", err)
		return
	}

	OK(c, nil)
}

// BrowseOrgInvite a user browses org invitation
// @Summary a user browses org invitation.
// @Produce json
// @Param id path string true "Invite link id"
// @Success 200 {object} apiserver.R{data=response.BrowseOrgInviteResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/inviteLinks/{id} [get]
func (h *APIHandler) BrowseOrgInvite(c *gin.Context) {
	var (
		err    error
		ctx    = c.Request.Context()
		linkID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if linkID == "" {
		err = errBizInvitationNotFound
		return
	}
	link, err := h.db.GetOrgInviteLinkByID(ctx, linkID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errBizInvitationNotFound
		return
	}
	if err != nil {
		err = fmt.Errorf("querying invite link: %w", err)
		return
	}

	// is the organization still present?
	org, err := h.db.GetOrganizationByID(ctx, link.OrgID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errBizInvitationNotFound
		return
	}
	if err != nil {
		err = fmt.Errorf("querying related org: %w", err)
		return
	}

	OK(c, response.BrowseOrgInviteResp{
		Organization: org,
	})
}

// ValidateInvite checks if current user can join the organization via the invite link
//
// It returns an error if the user has already been a member of the organization.
// @Summary checks if current user can join the organization via the invite link.
// @Description It returns an error if the user has already been a member of the organization.
// @Produce json
// @Param id path string true "Invite link id"
// @Success 200 {object} apiserver.R
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/inviteLinks/{id}/validation [get]
func (h *APIHandler) ValidateInvite(c *gin.Context) {
	var (
		err    error
		ctx    = c.Request.Context()
		linkID = c.Param("id")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if linkID == "" {
		err = errBizInvitationNotFound
		return
	}
	link, err := h.db.GetOrgInviteLinkByID(ctx, linkID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errBizInvitationNotFound
		return
	}
	if err != nil {
		err = fmt.Errorf("querying invite link: %w", err)
		return
	}

	// is the organization still present?
	_, err = h.db.GetOrganizationByID(ctx, link.OrgID)
	if errors.Is(err, sql.ErrNoRows) {
		err = errBizInvitationNotFound
		return
	}
	if err != nil {
		err = fmt.Errorf("querying related org: %w", err)
		return
	}

	userID := getSession(c).UserID
	_, err = h.db.GetOrgMemberByID(ctx, link.OrgID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		// relax
	} else if err == nil {
		err = errBizUserAlreadyOrgMember
		return
	} else {
		err = fmt.Errorf("querying user org membership: %w", err)
		return
	}

	OK(c, nil)
}

// ListOrgMembersByIDs list org members by org id and user ids, where the org must be visible to the user
// @Summary list org members by org id and user ids, where the org must be visible to the user
// @Produce json
// @Param id path int true "Organization ID"
// @Param   body body payload.ListOrgMembersByIDsReq true "HTTP body"
// @Success 200 {object} apiserver.R{data=[]model.OrganizationUser}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/{id}/members/findByIds [post]
func (h *APIHandler) ListOrgMembersByIDs(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req payload.ListOrgMembersByIDsReq
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	orgID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("binding org id: %w", err)
		return
	}
	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}

	userID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, userID, model.OwnerRef{
		OwnerType: model.OwnerTypeOrganization,
		OwnerID:   orgID,
	}, permission.OrgRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	members, err := h.db.ListOrgUsersByUserIDs(ctx, orgID, req.UserIDs...)
	if err != nil {
		err = fmt.Errorf("querying org member by user ids: %w", err)
		return
	}

	OK(c, members)
}

// SearchOrgMembers search org members by org id and username and emails, where the org must be visible to the user
// @Summary search org members by org id and username and emails, where the org must be visible to the user.
// @Produce json
// @Param id path int true "Organization ID"
// @Param q query string false "query for username or email"
// @Param size query int 5 "result size limit, max 10."
// @Success 200 {object} apiserver.R{data=[]model.OrganizationUser}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/orgs/{id}/members/search [get]
func (h *APIHandler) SearchOrgMembers(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
		req payload.SearchOrgMembersReq
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	orgID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("binding org id: %w", err)
		return
	}
	err = c.ShouldBindQuery(&req)
	if err != nil {
		err = fmt.Errorf("binding query: %w", err)
		return
	}

	const defaultSearchSize = 5
	if req.Size == 0 {
		req.Size = defaultSearchSize
	}

	userID := getSession(c).UserID

	err = h.enforcer.EnsurePermissions(ctx, userID, model.OwnerRef{
		OwnerType: model.OwnerTypeOrganization,
		OwnerID:   orgID,
	}, permission.OrgRead)
	if err != nil {
		err = fmt.Errorf("ensuring permissions: %w", err)
		_ = c.Error(err)
		err = errNoPermissionError
		return
	}

	members, err := h.db.SearchOrgUsers(ctx, orgID, req.Query, req.Size)
	if err != nil {
		err = fmt.Errorf("searching org members: %w", err)
		return
	}

	OK(c, members)
}
